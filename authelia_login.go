// Package handlers provides HTTP request handlers for the API Gateway.
//
// This file implements the Login handler for Authelia authentication.
//
// Associated Frontend Files:
//   - web/app/src/hooks/useAuth.ts (login function - POST /auth/login)
//   - web/app/src/pages/LoginPage.tsx (login form submission)
//   - web/app/src/lib/api.ts (apiClient.post for login)
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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

	// Extract username from email (e.g., admin@ugjb.com -> admin)
	// Authelia uses username, not email, for authentication
	username := req.Email
	if idx := strings.Index(req.Email, "@"); idx > 0 {
		username = req.Email[:idx]
	}

	// Convert to Authelia format
	autheliaReq := autheliaFirstFactorRequest{
		Username:       username,
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
		// Forward session cookies to client
		for _, cookie := range resp.Cookies() {
			if cookie.Name == h.config.Authelia.SessionCookieName {
				cookie.Domain = h.config.Authelia.SessionDomain
				cookie.Path = "/"
				cookie.HttpOnly = true
				cookie.Secure = c.Request.TLS != nil
				cookie.SameSite = http.SameSiteLaxMode
			}
			http.SetCookie(c.Writer, cookie)
		}

		// Extract username from email for user info
		username := req.Email
		if idx := strings.Index(req.Email, "@"); idx > 0 {
			username = req.Email[:idx]
		}

		// Generate JWT token for API authentication
		expiresAt := time.Now().Add(h.config.JWTExpiration)
		claims := &Claims{
			UserID: username,
			Email:  req.Email,
			Roles:  []string{"user"},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expiresAt),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    "ugjb-api-gateway",
				Subject:   username,
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(h.config.JWTSecret))
		if err != nil {
			h.logger.Error("Failed to generate JWT token", zap.Error(err))
			sendInternalError(c)
			return
		}

		h.logger.Info("User logged in successfully", zap.String("email", req.Email))

		// Return response compatible with frontend expectations
		c.JSON(http.StatusOK, gin.H{
			"status":     "OK",
			"token":      tokenString,
			"expires_at": expiresAt.UTC().Format(time.RFC3339),
			"user": gin.H{
				"id":    username,
				"name":  username,
				"email": req.Email,
				"roles": []string{"user"},
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
