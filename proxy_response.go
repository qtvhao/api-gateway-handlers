// Package handlers provides HTTP request handlers for the API Gateway.
//
// This file contains response modification and path rewriting proxy functionality.
//
// Associated Frontend Files:
//   - web/app/src/lib/api.ts (apiClient - all API calls proxied through gateway)
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// proxyRequestWithPathRewrite proxies a request and rewrites URLs in responses
func (p *ProxyHandler) proxyRequestWithPathRewrite(c *gin.Context, targetURL, targetPath, pathPrefix string) {
	target, err := url.Parse(targetURL)
	if err != nil {
		p.logger.Error("Failed to parse target URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Modify the request - disable compression to allow body rewriting
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = targetPath
		req.URL.RawQuery = c.Request.URL.RawQuery
		req.Host = target.Host

		// Disable compression so we can rewrite the body
		req.Header.Del("Accept-Encoding")

		// Forward other headers
		for key, values := range c.Request.Header {
			if key == "Accept-Encoding" {
				continue
			}
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
	}

	// Rewrite Location headers and HTML body URLs
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Rewrite Location header
		if location := resp.Header.Get("Location"); location != "" {
			if strings.HasPrefix(location, "/") && !strings.HasPrefix(location, pathPrefix) {
				resp.Header.Set("Location", pathPrefix+location)
			}
		}

		// Rewrite HTML body for text/html responses
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "text/html") {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}

			// Rewrite common URL patterns in HTML
			bodyStr := string(body)
			// Rewrite action="/..." and href="/..." and src="/..."
			bodyStr = strings.ReplaceAll(bodyStr, `action="/`, `action="`+pathPrefix+`/`)
			bodyStr = strings.ReplaceAll(bodyStr, `href="/`, `href="`+pathPrefix+`/`)
			bodyStr = strings.ReplaceAll(bodyStr, `src="/`, `src="`+pathPrefix+`/`)

			// Update body and content length
			newBody := []byte(bodyStr)
			resp.Body = io.NopCloser(strings.NewReader(bodyStr))
			resp.ContentLength = int64(len(newBody))
			resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
		}

		return nil
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
