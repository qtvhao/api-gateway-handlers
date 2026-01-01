// Package handlers provides authentication helper functions for the API Gateway.
//
// Associated Frontend Files:
//   - web/app/src/pages/LoginPage.tsx (login form and authentication flow)
//   - web/app/src/components/auth/ProtectedRoute.tsx (route protection with JWT)
//   - web/app/src/components/layout/Header.tsx (logout functionality)
//   - web/app/src/lib/api.ts (apiClient - JWT token management)
package handlers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ugjb/api-gateway/config"
	"github.com/ugjb/api-gateway/db"
	"go.uber.org/zap"
)

// authenticateUser validates user credentials against the user store
// Returns nil if authentication fails
func authenticateUser(cfg *config.Config, logger *zap.Logger, email, password string) *UserInfo {
	if email == "" || password == "" {
		return nil
	}

	// Validate credentials against configured users
	user, valid := validateCredentials(cfg, email, password)
	if !valid {
		return nil
	}

	return user
}

// validateCredentials checks email/password against the user store
// Production: Integrate with identity provider (LDAP, OAuth, OIDC) or database
func validateCredentials(cfg *config.Config, email, password string) (*UserInfo, bool) {
	// First, check users from SQLite database
	dbUser, err := db.ValidateCredentials(email, password)
	if err == nil && dbUser != nil {
		return &UserInfo{
			ID:    dbUser.ID,
			Name:  dbUser.Name,
			Email: dbUser.Email,
			Roles: dbUser.Roles,
		}, true
	}

	// Fallback to configured users from environment
	// Format: USER_CREDENTIALS="email1:password1:name1,email2:password2:name2"
	users := cfg.Auth.Users

	for _, user := range users {
		if user.Email == email && user.Password == password {
			return &UserInfo{
				ID:    user.ID,
				Name:  user.Name,
				Email: user.Email,
				Roles: determineRoles(cfg, email),
			}, true
		}
	}

	return nil, false
}

// generateToken creates a new JWT token for the user
func generateToken(cfg *config.Config, logger *zap.Logger, user *UserInfo) (string, time.Time, error) {
	// Token expires in 24 hours
	expiresAt := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Roles:  user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "ugjb-api-gateway",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Use configured JWT secret, fallback to default for development
	secret := cfg.JWTSecret
	if secret == "" {
		secret = "ugjb-development-secret-change-in-production"
		logger.Warn("Using default JWT secret - set JWT_SECRET environment variable in production")
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// determineRoles assigns roles based on email
func determineRoles(cfg *config.Config, email string) []string {
	// Default roles from config
	roles := make([]string, len(cfg.Auth.DefaultRoles))
	copy(roles, cfg.Auth.DefaultRoles)

	// Admin users from config
	if isAdminEmail(cfg, email) {
		roles = append(roles, "admin", "hr_manager")
	}

	// HR managers
	if containsAny(email, []string{"hr@", "hr_"}) {
		roles = append(roles, "hr_manager")
	}

	// Project managers
	if containsAny(email, []string{"pm@", "pm_", "project"}) {
		roles = append(roles, "project_manager")
	}

	return roles
}

// isAdminEmail checks if the email is in the configured admin list
func isAdminEmail(cfg *config.Config, email string) bool {
	for _, adminEmail := range cfg.Auth.AdminEmails {
		if email == adminEmail {
			return true
		}
	}
	return false
}

// extractNameFromEmail extracts a formatted name from an email address
func extractNameFromEmail(email string) string {
	// Extract name from email (e.g., "john.doe@example.com" -> "John Doe")
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	if atIndex == -1 {
		return email
	}

	name := email[:atIndex]
	// Replace dots and underscores with spaces, capitalize words
	result := make([]byte, 0, len(name))
	capitalizeNext := true
	for _, c := range name {
		if c == '.' || c == '_' || c == '-' {
			result = append(result, ' ')
			capitalizeNext = true
		} else if capitalizeNext && c >= 'a' && c <= 'z' {
			result = append(result, byte(c-32)) // Uppercase
			capitalizeNext = false
		} else {
			result = append(result, byte(c))
			capitalizeNext = false
		}
	}
	return string(result)
}

// containsAny checks if string s contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
	}
	return false
}

// updateUserPassword updates the password for a user in the database
func updateUserPassword(cfg *config.Config, logger *zap.Logger, email, newPassword string) error {
	return db.UpdatePassword(email, newPassword)
}
