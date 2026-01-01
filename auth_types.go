// Package handlers provides authentication type definitions for the API Gateway.
//
// Associated Frontend Files:
//   - web/app/src/hooks/useAuth.ts (authentication hook)
//   - web/app/src/lib/api.ts (API client with auth token)
//   - web/app/src/pages/LoginPage.tsx (login form)
package handlers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=1"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

// UserInfo represents user information in the response
type UserInfo struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// Claims represents JWT claims
type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// ChangePasswordRequest represents the change password request body
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=1"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=8"`
}
