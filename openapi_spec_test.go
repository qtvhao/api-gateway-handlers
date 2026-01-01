package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSwaggerDocJSONEndpoint verifies that the OpenAPI JSON spec is accessible
func TestSwaggerDocJSONEndpoint(t *testing.T) {
	router := setupSwaggerRouter()

	req, _ := http.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify content type is JSON
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type 'application/json; charset=utf-8', got '%s'", contentType)
	}

	// Verify body is valid JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &jsonData); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}
}

// TestOpenAPIJSONRedirect verifies that /openapi.json redirects to /swagger/doc.json
func TestOpenAPIJSONRedirect(t *testing.T) {
	router := setupSwaggerRouter()

	req, _ := http.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected status %d (Moved Permanently), got %d", http.StatusMovedPermanently, w.Code)
	}

	// Verify Location header points to swagger doc.json
	location := w.Header().Get("Location")
	expectedLocation := "/swagger/doc.json"
	if location != expectedLocation {
		t.Errorf("Expected Location header '%s', got '%s'", expectedLocation, location)
	}
}

// TestOpenAPISpecStructure verifies that the OpenAPI spec has required fields
func TestOpenAPISpecStructure(t *testing.T) {
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

	// Verify required OpenAPI fields exist
	requiredFields := []string{"swagger", "info", "paths"}
	for _, field := range requiredFields {
		if _, exists := spec[field]; !exists {
			t.Errorf("OpenAPI spec missing required field: %s", field)
		}
	}

	// Verify swagger version is 2.0
	if swagger, ok := spec["swagger"].(string); ok {
		if swagger != "2.0" {
			t.Errorf("Expected swagger version '2.0', got '%s'", swagger)
		}
	} else {
		t.Error("swagger field is not a string")
	}

	// Verify info object has required fields
	if info, ok := spec["info"].(map[string]interface{}); ok {
		infoRequiredFields := []string{"title", "version", "description"}
		for _, field := range infoRequiredFields {
			if _, exists := info[field]; !exists {
				t.Errorf("OpenAPI spec info missing required field: %s", field)
			}
		}

		// Verify title
		if title, ok := info["title"].(string); ok {
			if title != "UGJB API Gateway" {
				t.Errorf("Expected title 'UGJB API Gateway', got '%s'", title)
			}
		}

		// Verify version
		if version, ok := info["version"].(string); ok {
			if version != "1.0.0" {
				t.Errorf("Expected version '1.0.0', got '%s'", version)
			}
		}
	} else {
		t.Error("info field is not an object")
	}
}

// TestOpenAPISecurityDefinitions verifies that security definitions are present
func TestOpenAPISecurityDefinitions(t *testing.T) {
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

	// Verify securityDefinitions exist
	securityDefs, ok := spec["securityDefinitions"].(map[string]interface{})
	if !ok {
		t.Fatal("securityDefinitions field is missing or not an object")
	}

	// Verify BearerAuth is defined
	if _, exists := securityDefs["BearerAuth"]; !exists {
		t.Error("BearerAuth security definition not found")
	}
}

// TestOpenAPIDefinitions verifies that model definitions are present
func TestOpenAPIDefinitions(t *testing.T) {
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

	// Verify definitions exist
	definitions, ok := spec["definitions"].(map[string]interface{})
	if !ok {
		t.Fatal("definitions field is missing or not an object")
	}

	// Verify expected model definitions exist
	expectedDefinitions := []string{
		"handlers.LoginRequest",
		"handlers.LoginResponse",
		"handlers.UserInfo",
	}

	for _, def := range expectedDefinitions {
		if _, exists := definitions[def]; !exists {
			t.Errorf("Expected definition '%s' not found in OpenAPI spec", def)
		}
	}
}
