// Package handlers provides HTTP request handlers for the API Gateway.
//
// This file contains type definitions for Authelia authentication handlers.
//
// Architecture:
//   Browser -> API Gateway (:8080) -> Authelia (:9091 internal) -> Redis (sessions)
//
// See: agent/docs/network-topology/api-gateway-topology.mmd
package handlers

// AutheliaLoginRequest represents the login request body (matches frontend)
type AutheliaLoginRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=1"`
	KeepMeLoggedIn bool   `json:"keepMeLoggedIn"`
	TargetURL      string `json:"targetURL,omitempty"`
}

// autheliaFirstFactorRequest is the internal format for Authelia /api/firstfactor
type autheliaFirstFactorRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	KeepMeLoggedIn bool   `json:"keepMeLoggedIn"`
	TargetURL      string `json:"targetURL,omitempty"`
}

// AutheliaLoginResponse represents the login response
type AutheliaLoginResponse struct {
	Status string `json:"status"`
	User   struct {
		Username string   `json:"username"`
		Name     string   `json:"name"`
		Email    string   `json:"email"`
		Groups   []string `json:"groups"`
	} `json:"user,omitempty"`
	Redirect string `json:"redirect,omitempty"`
}

// autheliaFirstFactorResponse is the internal format for Authelia firstfactor response
type autheliaFirstFactorResponse struct {
	Status string `json:"status"`
	Data   struct {
		Redirect string `json:"redirect,omitempty"`
	} `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// autheliaUserInfo is internal type matching middleware.AutheliaUserInfo
type autheliaUserInfo struct {
	Username string
	Name     string
	Email    string
	Groups   []string
}
