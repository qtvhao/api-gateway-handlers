// Package handlers provides HTTP request handlers for the API Gateway.
//
// Associated Frontend Files:
//   - web/app/src/lib/api.ts (apiClient - all API calls proxied through gateway)
//   - web/app/src/pages/* (all page components making API requests)
package handlers

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ugjb/api-gateway/config"
	"go.uber.org/zap"
)

// ProxyHandler handles proxying requests to backend services
type ProxyHandler struct {
	config *config.Config
	logger *zap.Logger
}

// NewProxyHandler creates a new ProxyHandler
func NewProxyHandler(cfg *config.Config, logger *zap.Logger) *ProxyHandler {
	return &ProxyHandler{
		config: cfg,
		logger: logger,
	}
}

// ProxyToService returns a handler that proxies to a backend service
func (p *ProxyHandler) ProxyToService(serviceName, targetPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceURL := p.getServiceURL(serviceName)
		if serviceURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Service %s not configured", serviceName),
			})
			return
		}

		p.proxyRequest(c, serviceURL, targetPath)
	}
}

// ProxyToExternalService proxies to external services
func (p *ProxyHandler) ProxyToExternalService(serviceName, targetPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceURL := p.getServiceURL(serviceName)
		if serviceURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("External service %s not configured", serviceName),
			})
			return
		}

		p.proxyRequest(c, serviceURL, targetPath)
	}
}

// ProxyWithWebSocket handles proxying including WebSocket connections
func (p *ProxyHandler) ProxyWithWebSocket(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Return JSON 404 for undefined API routes (don't proxy to frontend)
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "API endpoint not found",
					"path":    c.Request.URL.Path,
				},
			})
			return
		}

		serviceURL := p.getServiceURL(serviceName)
		if serviceURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Service %s not configured", serviceName),
			})
			return
		}

		// Check if this is a WebSocket upgrade request
		if c.GetHeader("Upgrade") == "websocket" {
			p.proxyWebSocket(c, serviceURL)
			return
		}

		p.proxyRequest(c, serviceURL, c.Request.URL.Path)
	}
}

// proxyRequest proxies a regular HTTP request
func (p *ProxyHandler) proxyRequest(c *gin.Context, targetURL, targetPath string) {
	target, err := url.Parse(targetURL)
	if err != nil {
		p.logger.Error("Failed to parse target URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Build the target path
		if strings.Contains(targetPath, ":id") {
			// Replace :id with the actual parameter
			id := c.Param("id")
			targetPath = strings.Replace(targetPath, ":id", id, 1)
		}

		// Preserve query parameters
		req.URL.Path = targetPath
		req.URL.RawQuery = c.Request.URL.RawQuery
		req.Host = target.Host

		// Forward headers (use Set to prevent header accumulation causing 431 errors)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				req.Header.Set(key, values[0])
				// Add remaining values if multiple exist
				for _, value := range values[1:] {
					req.Header.Add(key, value)
				}
			}
		}

		// Add forwarding headers
		req.Header.Set("X-Forwarded-For", c.ClientIP())
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Header.Set("X-Real-IP", c.ClientIP())

		// Forward user info from auth middleware
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok && uid != "" {
				req.Header.Set("X-User-ID", uid)
			}
		}
		if email, exists := c.Get("email"); exists {
			if e, ok := email.(string); ok && e != "" {
				req.Header.Set("X-User-Email", e)
			}
		}
	}

	// Handle errors
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		p.logger.Error("Proxy error", zap.Error(err), zap.String("target", targetURL))
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "Service unavailable",
			"details": err.Error(),
		})
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

// ProxyToAuthelia returns a handler that proxies requests to internal Authelia
// Authelia is never exposed publicly - only accessible via internal Docker network
func (p *ProxyHandler) ProxyToAuthelia() gin.HandlerFunc {
	return func(c *gin.Context) {
		autheliaURL := p.config.Authelia.InternalURL
		if autheliaURL == "" {
			p.logger.Error("Authelia URL not configured")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"code":    "AUTH_SERVICE_UNAVAILABLE",
					"message": "Authentication service not configured",
				},
			})
			return
		}

		// Extract the path after /api/oidc or /api/auth
		path := c.Param("path")
		if path == "" {
			path = "/"
		}

		targetPath := "/api/oidc" + path
		p.proxyRequest(c, autheliaURL, targetPath)
	}
}

// proxyWebSocket handles WebSocket proxy
func (p *ProxyHandler) proxyWebSocket(c *gin.Context, targetURL string) {
	target, err := url.Parse(targetURL)
	if err != nil {
		p.logger.Error("Failed to parse WebSocket target URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Create WebSocket URL
	wsURL := fmt.Sprintf("ws://%s%s", target.Host, c.Request.URL.Path)
	if c.Request.URL.RawQuery != "" {
		wsURL += "?" + c.Request.URL.RawQuery
	}

	p.logger.Info("WebSocket proxy", zap.String("target", wsURL))

	// For now, return an error as WebSocket proxying requires more complex handling
	// In production, you'd use a proper WebSocket proxy library
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "WebSocket proxying not implemented in this version",
	})
}
