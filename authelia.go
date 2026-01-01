// Package handlers provides HTTP request handlers for the API Gateway.
//
// This file implements Authelia authentication proxy handlers.
// Authelia runs INTERNALLY behind the API Gateway and is never exposed publicly.
//
// Associated Frontend Files:
//   - web/app/src/hooks/useAuth.ts (login, logout, session management)
//   - web/app/src/pages/LoginPage.tsx (login form UI)
//   - web/app/src/lib/api.ts (apiClient - auth token handling)
//   - web/app/src/components/auth/ProtectedRoute.tsx (auth state checks)
//
// Architecture:
//   Browser -> API Gateway (:8080) -> Authelia (:9091 internal) -> Redis (sessions)
//
// The gateway proxies auth routes to Authelia's internal endpoints:
//   - POST /api/v1/auth/login -> Authelia /api/firstfactor
//   - POST /api/v1/auth/logout -> Authelia /api/logout
//   - GET /api/v1/auth/session -> Authelia /api/user/info
//
// Related files:
//   - authelia_types.go: Type definitions for requests/responses
//   - authelia_helpers.go: Helper functions for responses and cookies
//   - authelia_login.go: Login handler implementation
//   - authelia_logout.go: Logout handler implementation
//
// See: agent/docs/network-topology/api-gateway-topology.mmd
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ugjb/api-gateway/config"
	"go.uber.org/zap"
)

// AutheliaHandler handles authentication requests by proxying to internal Authelia
type AutheliaHandler struct {
	config *config.Config
	logger *zap.Logger
	client *http.Client
}

// NewAutheliaHandler creates a new AutheliaHandler
func NewAutheliaHandler(cfg *config.Config, logger *zap.Logger) *AutheliaHandler {
	return &AutheliaHandler{
		config: cfg,
		logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetSession returns the current user's session information
// @Summary Get current session
// @Description Returns the authenticated user's session information from Authelia
// @Tags Authentication
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Session information"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 502 {object} map[string]interface{} "Auth service unavailable"
// @Router /api/v1/auth/session [get]
func (h *AutheliaHandler) GetSession(c *gin.Context) {
	// Call Authelia /api/user/info (internal network only)
	autheliaURL := h.config.Authelia.InternalURL + "/api/user/info"
	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), "GET", autheliaURL, nil)
	if err != nil {
		h.logger.Error("Failed to create Authelia session request", zap.Error(err))
		sendInternalError(c)
		return
	}

	// Forward session cookie
	if cookie, err := c.Cookie(h.config.Authelia.SessionCookieName); err == nil {
		proxyReq.AddCookie(&http.Cookie{
			Name:  h.config.Authelia.SessionCookieName,
			Value: cookie,
		})
	} else {
		sendUnauthorizedError(c)
		return
	}

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.Error("Authelia session request failed", zap.Error(err))
		sendBadGatewayError(c)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Failed to read Authelia session response", zap.Error(err))
		sendInternalError(c)
		return
	}

	if resp.StatusCode != http.StatusOK {
		sendUnauthorizedError(c)
		return
	}

	// Parse and forward Authelia response
	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		h.logger.Error("Failed to parse Authelia session response", zap.Error(err))
		sendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, userInfo)
}

// GetCurrentUser returns the authenticated user's information (alias for GetSession)
// This maintains compatibility with existing frontend code
// @Summary Get current user
// @Description Returns the authenticated user's information
// @Tags Authentication
// @Accept json
// @Produce json
// @Security SessionCookie
// @Success 200 {object} UserInfo "Current user information"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /api/v1/auth/me [get]
func (h *AutheliaHandler) GetCurrentUser(c *gin.Context) {
	// Get user from context (set by AutheliaForwardAuth middleware)
	user, exists := c.Get("authelia_user")
	if !exists {
		sendUnauthorizedError(c)
		return
	}

	autheliaUser := user.(*autheliaUserInfo)
	c.JSON(http.StatusOK, UserInfo{
		ID:    autheliaUser.Username, // Use username as ID
		Name:  autheliaUser.Name,
		Email: autheliaUser.Email,
		Roles: autheliaUser.Groups,
	})
}
