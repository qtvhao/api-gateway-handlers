// Package handlers provides HTTP request handlers for the API Gateway.
//
// This file implements the Login handler for Authelia authentication.
//
// Architecture:
//   Browser -> API Gateway (:8080) -> Authelia (:9091 internal) -> Redis (sessions)
//
// See: agent/docs/network-topology/api-gateway-topology.mmd
package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Login handles user authentication by proxying to internal Authelia
// @Summary User login
// @Description Authenticate user with email and password via Authelia
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body AutheliaLoginRequest true "Login credentials"
// @Success 200 {object} AutheliaLoginResponse "Successful authentication"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Failure 502 {object} map[string]interface{} "Auth service unavailable"
// @Router /api/v1/auth/login [post]
func (h *AutheliaHandler) Login(c *gin.Context) {
	var req AutheliaLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid login request", zap.Error(err))
		sendInvalidRequestError(c)
		return
	}

	// Convert to Authelia format (email -> username)
	autheliaReq := autheliaFirstFactorRequest{
		Username:       req.Email, // Authelia uses username, we accept email
		Password:       req.Password,
		KeepMeLoggedIn: req.KeepMeLoggedIn,
		TargetURL:      req.TargetURL,
	}

	reqBody, err := json.Marshal(autheliaReq)
	if err != nil {
		h.logger.Error("Failed to marshal Authelia request", zap.Error(err))
		sendInternalError(c)
		return
	}

	// Call Authelia /api/firstfactor (internal network only)
	autheliaURL := h.config.Authelia.InternalURL + "/api/firstfactor"
	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", autheliaURL, bytes.NewReader(reqBody))
	if err != nil {
		h.logger.Error("Failed to create Authelia request", zap.Error(err))
		sendInternalError(c)
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("X-Forwarded-For", c.ClientIP())
	proxyReq.Header.Set("X-Forwarded-Proto", getScheme(c))
	proxyReq.Header.Set("X-Forwarded-Host", c.Request.Host)

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.Error("Authelia login request failed", zap.Error(err))
		sendBadGatewayError(c)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Failed to read Authelia response", zap.Error(err))
		sendInternalError(c)
		return
	}

	// Parse Authelia response
	var autheliaResp autheliaFirstFactorResponse
	if err := json.Unmarshal(body, &autheliaResp); err != nil {
		h.logger.Error("Failed to parse Authelia response",
			zap.Error(err),
			zap.String("body", string(body)),
		)
		sendInternalError(c)
		return
	}

	h.handleLoginResponse(c, resp, &req, &autheliaResp, body)
}

// handleLoginResponse processes the Authelia login response
func (h *AutheliaHandler) handleLoginResponse(c *gin.Context, resp *http.Response, req *AutheliaLoginRequest, autheliaResp *autheliaFirstFactorResponse, body []byte) {
	switch resp.StatusCode {
	case http.StatusOK:
		// Forward Set-Cookie headers from Authelia to client
		for _, cookie := range resp.Cookies() {
			// Adjust cookie domain if needed
			if cookie.Name == h.config.Authelia.SessionCookieName {
				cookie.Domain = h.config.Authelia.SessionDomain
				cookie.Path = "/"
				cookie.HttpOnly = true
				cookie.Secure = c.Request.TLS != nil
				cookie.SameSite = http.SameSiteLaxMode
			}
			http.SetCookie(c.Writer, cookie)
		}

		h.logger.Info("User logged in successfully", zap.String("email", req.Email))

		c.JSON(http.StatusOK, gin.H{
			"status": "OK",
			"user": gin.H{
				"email": req.Email,
			},
			"redirect": autheliaResp.Data.Redirect,
		})

	case http.StatusUnauthorized:
		h.logger.Warn("Authentication failed", zap.String("email", req.Email))
		sendInvalidCredentialsError(c)

	default:
		h.logger.Error("Unexpected Authelia response",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		sendAuthServiceError(c)
	}
}
