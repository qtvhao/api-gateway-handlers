// Package handlers provides HTTP request handlers for the API Gateway.
//
// Associated Frontend Files:
//   - web/app/src/hooks/useAuth.ts (authentication hook)
//   - web/app/src/lib/api.ts (API client with auth token)
//   - web/app/src/pages/LoginPage.tsx (login form)
//
// Type definitions: auth_types.go
// Helper functions: auth_helpers.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ugjb/api-gateway/config"
	"go.uber.org/zap"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	config *config.Config
	logger *zap.Logger
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(cfg *config.Config, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		config: cfg,
		logger: logger,
	}
}

// Login handles user authentication and returns a JWT token
// @Summary User login
// @Description Authenticate user with email and password, returns JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse "Successful authentication"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid login request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Authenticate user credentials via authenticateUser method
	// Production integration: See docs/deployment-architecture/security-compliance/ for identity provider setup
	user := authenticateUser(h.config, h.logger, req.Email, req.Password)
	if user == nil {
		h.logger.Warn("Authentication failed", zap.String("email", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
		})
		return
	}

	// Generate JWT token
	token, expiresAt, err := generateToken(h.config, h.logger, user)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate authentication token",
		})
		return
	}

	h.logger.Info("User logged in successfully", zap.String("email", user.Email))

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      *user,
	})
}

// RefreshToken handles token refresh requests
// @Summary Refresh token
// @Description Refresh an existing JWT token to extend session
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LoginResponse "New token issued"
// @Failure 401 {object} map[string]interface{} "Invalid or expired token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Get user info from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token",
		})
		return
	}

	email, _ := c.Get("email")
	roles, _ := c.Get("roles")

	user := &UserInfo{
		ID:    userID.(string),
		Email: email.(string),
		Roles: roles.([]string),
	}

	// Extract name from email for demo purposes
	user.Name = extractNameFromEmail(user.Email)

	// Generate new token
	token, expiresAt, err := generateToken(h.config, h.logger, user)
	if err != nil {
		h.logger.Error("Failed to refresh token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      *user,
	})
}

// Logout handles user logout (token invalidation)
// @Summary User logout
// @Description Invalidate the current user session
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Logged out successfully"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a production environment, you would:
	// 1. Add the token to a blacklist (Redis/database)
	// 2. Clear any server-side sessions
	// For now, we just return success (client handles token removal)
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// Me returns the current user's information
// @Summary Get current user
// @Description Returns the authenticated user's information
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserInfo "Current user information"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}

	email, _ := c.Get("email")
	roles, _ := c.Get("roles")

	c.JSON(http.StatusOK, UserInfo{
		ID:    userID.(string),
		Name:  extractNameFromEmail(email.(string)),
		Email: email.(string),
		Roles: roles.([]string),
	})
}

// ChangePassword handles password change requests
// Associated Frontend: web/app/src/pages/settings/SecuritySettings.tsx
// @Summary Change password
// @Description Change the authenticated user's password
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Password change request"
// @Success 200 {object} map[string]interface{} "Password changed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or passwords don't match"
// @Failure 401 {object} map[string]interface{} "Not authenticated or invalid current password"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid change password request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body. New password must be at least 8 characters.",
		})
		return
	}

	// Validate passwords match
	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "New password and confirmation do not match",
		})
		return
	}

	// Get user info from context (set by auth middleware)
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}

	// Verify current password
	user := authenticateUser(h.config, h.logger, email.(string), req.CurrentPassword)
	if user == nil {
		h.logger.Warn("Password change failed - invalid current password", zap.String("email", email.(string)))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Current password is incorrect",
		})
		return
	}

	// Update password in database
	if err := updateUserPassword(h.config, h.logger, email.(string), req.NewPassword); err != nil {
		h.logger.Error("Failed to update password", zap.Error(err), zap.String("email", email.(string)))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update password",
		})
		return
	}

	h.logger.Info("Password changed successfully", zap.String("email", email.(string)))

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}
