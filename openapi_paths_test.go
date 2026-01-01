package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestOpenAPIPathsExist verifies that expected API paths are documented
func TestOpenAPIPathsExist(t *testing.T) {
	router := setupSwaggerRouter()

	req, _ := http.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("paths field is not an object")
	}

	// Verify expected paths exist
	expectedPaths := []string{
		"/health",
		"/api/v1/auth/login",
		"/api/v1/auth/me",
		"/api/v1/employees",
	}

	for _, path := range expectedPaths {
		if _, exists := paths[path]; !exists {
			t.Errorf("Expected path '%s' not found in OpenAPI spec", path)
		}
	}
}

// TestOpenAPIAuthEndpointMethods verifies that auth endpoints have correct HTTP methods
func TestOpenAPIAuthEndpointMethods(t *testing.T) {
	router := setupSwaggerRouter()

	req, _ := http.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("paths field is not an object")
	}

	// Test /api/v1/auth/login has POST method
	if loginPath, exists := paths["/api/v1/auth/login"].(map[string]interface{}); exists {
		if _, hasPost := loginPath["post"]; !hasPost {
			t.Error("/api/v1/auth/login should have POST method")
		}
	}

	// Test /api/v1/auth/me has GET method
	if mePath, exists := paths["/api/v1/auth/me"].(map[string]interface{}); exists {
		if _, hasGet := mePath["get"]; !hasGet {
			t.Error("/api/v1/auth/me should have GET method")
		}
	}

	// Test /api/v1/auth/logout has POST method
	if logoutPath, exists := paths["/api/v1/auth/logout"].(map[string]interface{}); exists {
		if _, hasPost := logoutPath["post"]; !hasPost {
			t.Error("/api/v1/auth/logout should have POST method")
		}
	}
}
