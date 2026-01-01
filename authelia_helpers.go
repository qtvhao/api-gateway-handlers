// Package handlers provides HTTP request handlers for the API Gateway.
//
// This file contains helper functions for Authelia authentication handlers.
//
// Architecture:
//   Browser -> API Gateway (:8080) -> Authelia (:9091 internal) -> Redis (sessions)
//
// See: agent/docs/network-topology/api-gateway-topology.mmd
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// clearSessionCookie clears the Authelia session cookie
func (h *AutheliaHandler) clearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.config.Authelia.SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   h.config.Authelia.SessionDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

// getScheme determines the request scheme (http/https)
func getScheme(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
}

// sendInternalError sends a standardized internal error response
func sendInternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Internal server error",
		},
	})
}

// sendUnauthorizedError sends a standardized unauthorized error response
func sendUnauthorizedError(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"code":    "UNAUTHORIZED",
			"message": "Not authenticated",
		},
	})
}

// sendBadGatewayError sends a standardized bad gateway error response
func sendBadGatewayError(c *gin.Context) {
	c.JSON(http.StatusBadGateway, gin.H{
		"error": gin.H{
			"code":    "AUTH_SERVICE_UNAVAILABLE",
			"message": "Authentication service unavailable",
		},
	})
}

// sendInvalidCredentialsError sends a standardized invalid credentials error response
func sendInvalidCredentialsError(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"code":    "INVALID_CREDENTIALS",
			"message": "Invalid email or password",
		},
	})
}

// sendInvalidRequestError sends a standardized invalid request error response
func sendInvalidRequestError(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Invalid request body",
		},
	})
}

// sendAuthServiceError sends a standardized auth service error response
func sendAuthServiceError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":    "AUTH_SERVICE_ERROR",
			"message": "Authentication service error",
		},
	})
}
