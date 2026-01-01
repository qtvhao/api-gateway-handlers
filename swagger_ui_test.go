package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/ugjb/api-gateway/docs"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupSwaggerRouter creates a router with swagger endpoints for testing
func setupSwaggerRouter() *gin.Engine {
	router := gin.New()

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// OpenAPI JSON redirect endpoint
	router.GET("/openapi.json", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/doc.json")
	})

	return router
}

// TestSwaggerUIEndpoint verifies that the Swagger UI is accessible
func TestSwaggerUIEndpoint(t *testing.T) {
	router := setupSwaggerRouter()

	req, _ := http.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify content type is HTML
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/html; charset=utf-8', got '%s'", contentType)
	}

	// Verify body contains Swagger UI elements
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty response body for Swagger UI")
	}
}

// TestSwaggerStaticAssets verifies that Swagger static assets are served
func TestSwaggerStaticAssets(t *testing.T) {
	router := setupSwaggerRouter()

	// Test swagger-ui.css
	req, _ := http.NewRequest(http.MethodGet, "/swagger/swagger-ui.css", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for swagger-ui.css, got %d", http.StatusOK, w.Code)
	}

	// Test swagger-ui-bundle.js
	req, _ = http.NewRequest(http.MethodGet, "/swagger/swagger-ui-bundle.js", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for swagger-ui-bundle.js, got %d", http.StatusOK, w.Code)
	}
}
