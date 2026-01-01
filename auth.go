// Package handlers provides HTTP request handlers for the API Gateway.
//
// DEPRECATED: This file is deprecated. Use authelia.go instead.
//
// Associated Frontend Files:
//   - web/app/src/hooks/useAuth.ts (authentication hook)
//   - web/app/src/pages/LoginPage.tsx (login form)
//   - web/app/src/lib/api.ts (auth token management)
//
// Authentication is now handled by Authelia (internal only - never exposed publicly).
// See: handlers/authelia.go for the new implementation.
//
// Architecture:
//   Browser -> API Gateway (:8080) -> Authelia (:9091 internal) -> Redis (sessions)
//
// Migration:
//   - AuthHandler.Login -> AutheliaHandler.Login
//   - AuthHandler.Logout -> AutheliaHandler.Logout
//   - AuthHandler.Me -> AutheliaHandler.GetCurrentUser
//   - AuthHandler.RefreshToken -> Not needed (Authelia manages sessions)
//   - AuthHandler.ChangePassword -> Authelia portal handles this
//
// See: agent/docs/network-topology/api-gateway-topology.mmd
package handlers

// NOTE: The AuthHandler struct and its methods have been REMOVED.
//
// The old implementation violated architecture decisions:
// - Called authenticateUser() which validated credentials in the gateway
// - Called generateToken() which issued JWTs in the gateway
// - Called updateUserPassword() which accessed the database directly
//
// These functions have been removed per ADR-0010.
// All authentication is now handled by Authelia via:
// - handlers/authelia.go (AutheliaHandler)
// - middleware/authelia.go (AutheliaForwardAuth)
//
// See: docs/ADRs/api/0010-reverse-proxy-gateway.md
