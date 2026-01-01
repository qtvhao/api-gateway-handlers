// Package handlers provides HTTP request handlers for the API Gateway.
//
// This file contains external service proxy handlers (Bugsink, DirectProxy).
//
// Associated Frontend Files:
//   - web/app/src/lib/api.ts (apiClient - error tracking via Bugsink)
package handlers

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProxyBugsink handles proxying to Bugsink (Sentry-compatible) error tracking
// Preserves original Host header for CSRF validation
func (p *ProxyHandler) ProxyBugsink() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceURL := p.getServiceURL("bugsink")
		if serviceURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Bugsink service not configured",
			})
			return
		}

		// Strip /sentry prefix when forwarding - Bugsink routes at /
		path := c.Param("path")
		if path == "" {
			path = "/"
		}

		target, err := url.Parse(serviceURL)
		if err != nil {
			p.logger.Error("Failed to parse target URL", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		// Preserve original Host header for CSRF validation
		originalHost := c.Request.Host

		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = path
			req.URL.RawQuery = c.Request.URL.RawQuery

			// Keep original Host header (critical for CSRF)
			req.Host = originalHost

			// Forward headers
			for key, values := range c.Request.Header {
				if len(values) > 0 {
					req.Header.Set(key, values[0])
					for _, value := range values[1:] {
						req.Header.Add(key, value)
					}
				}
			}

			req.Header.Set("X-Forwarded-For", c.ClientIP())
			req.Header.Set("X-Forwarded-Proto", "http")
			req.Header.Set("X-Real-IP", c.ClientIP())
			req.Header.Set("X-Forwarded-Host", originalHost)
		}

		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			p.logger.Error("Bugsink proxy error", zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{
				"error":   "Bugsink service unavailable",
				"details": err.Error(),
			})
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// DirectProxy directly proxies to a specific URL
func (p *ProxyHandler) DirectProxy(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, err := url.Parse(targetURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
			return
		}

		// Read the request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}

		// Create new request
		req, err := http.NewRequest(c.Request.Method, target.String()+c.Request.URL.Path, strings.NewReader(string(body)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
			return
		}

		// Copy headers (use Set to prevent header accumulation causing 431 errors)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				req.Header.Set(key, values[0])
				for _, value := range values[1:] {
					req.Header.Add(key, value)
				}
			}
		}

		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach service"})
			return
		}
		defer resp.Body.Close()

		// Copy response
		respBody, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
	}
}
