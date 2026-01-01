// Package handlers provides authentication helper functions for the API Gateway.
//
// IMPORTANT: Authentication is handled by Authelia (internal only).
// This file contains only utility functions, NOT credential validation.
//
// Architecture:
//   Browser -> API Gateway (:8080) -> Authelia (:9091 internal) -> Redis (sessions)
//
// Authelia handles:
//   - User authentication (login/logout)
//   - Session management (via Redis)
//   - Password management
//   - Multi-factor authentication
//
// The gateway:
//   - Proxies auth requests to internal Authelia
//   - Uses forward-auth to validate protected routes
//   - Forwards user info headers to backend services
//
// See: agent/docs/network-topology/api-gateway-topology.mmd
// See: middleware/authelia.go for forward-auth implementation
// See: handlers/authelia.go for auth proxy handlers
package handlers

// NOTE: The following functions have been REMOVED as they violated architecture decisions:
//
// - authenticateUser() - REMOVED: Gateway must not validate credentials directly
//   -> Use: Authelia /api/firstfactor (via handlers/authelia.go)
//
// - validateCredentials() - REMOVED: Gateway must not access user database
//   -> Use: Authelia manages users in its own configuration
//
// - generateToken() - REMOVED: Gateway must not issue tokens
//   -> Use: Authelia issues session cookies
//
// - determineRoles() - REMOVED: Gateway must not manage roles
//   -> Use: Authelia returns Remote-Groups header
//
// - updateUserPassword() - REMOVED: Gateway must not access database (ADR-0010)
//   -> Use: Authelia /api/user/info for password changes
//
// See ADR-0010: Reverse Proxy Gateway for External Integration
// The gateway should NOT access database directly.

// extractNameFromEmail extracts a formatted name from an email address
// This is a utility function that doesn't require Authelia
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
// This is a utility function that doesn't require Authelia
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
