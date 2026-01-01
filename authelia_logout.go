// Package handlers provides HTTP request handlers for the API Gateway.
//
// This file implements the Logout handler for Authelia authentication.
//
// Architecture:
//   Browser -> API Gateway (:8080) -> Authelia (:9091 internal) -> Redis (sessions)
//
// See: agent/docs/network-topology/api-gateway-topology.mmd
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logout handles user logout by proxying to internal Authelia
// @Summary User logout
// @Description Invalidate the current user session via Authelia
// @Tags Authentication
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Logged out successfully"
// @Failure 502 {object} map[string]interface{} "Auth service unavailable"
// @Router /api/v1/auth/logout [post]
func (h *AutheliaHandler) Logout(c *gin.Context) {
	// Call Authelia /api/logout (internal network only)
	autheliaURL := h.config.Authelia.InternalURL + "/api/logout"
	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", autheliaURL, nil)
	if err != nil {
		h.logger.Error("Failed to create Authelia logout request", zap.Error(err))
		sendInternalError(c)
		return
	}

	// Forward session cookie
	if cookie, err := c.Cookie(h.config.Authelia.SessionCookieName); err == nil {
		proxyReq.AddCookie(&http.Cookie{
			Name:  h.config.Authelia.SessionCookieName,
			Value: cookie,
		})
	}

	proxyReq.Header.Set("X-Forwarded-For", c.ClientIP())

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.Error("Authelia logout request failed", zap.Error(err))
		// Still clear the cookie on the client side
		h.clearSessionCookie(c)
		c.JSON(http.StatusOK, gin.H{
			"message": "Logged out",
		})
		return
	}
	defer resp.Body.Close()

	// Forward Set-Cookie headers (to clear the session)
	for _, cookie := range resp.Cookies() {
		http.SetCookie(c.Writer, cookie)
	}

	// Also explicitly clear the session cookie
	h.clearSessionCookie(c)

	h.logger.Info("User logged out")

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}
