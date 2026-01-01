// Package handlers provides HTTP request handlers for the API Gateway.
//
// Associated Frontend Files:
//   - None (internal health monitoring endpoints)
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	logger    *zap.Logger
	startTime time.Time
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		logger:    logger,
		startTime: time.Now(),
	}
}

// Health returns basic health status
// @Summary Health check
// @Description Returns the health status of the API Gateway
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Health status"
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "api-gateway",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready returns readiness status
// @Summary Readiness check
// @Description Returns the readiness status of the API Gateway
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Readiness status"
// @Router /health/ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"service":   "api-gateway",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Live returns liveness status
// @Summary Liveness check
// @Description Returns the liveness status of the API Gateway with uptime
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Liveness status with uptime"
// @Router /health/live [get]
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "live",
		"service":   "api-gateway",
		"uptime":    time.Since(h.startTime).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Liveness returns liveness status (alternate endpoint)
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"service":   "api-gateway",
		"uptime":    time.Since(h.startTime).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Readiness returns readiness status (alternate endpoint)
func (h *HealthHandler) Readiness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"service":   "api-gateway",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Status returns detailed status information
// @Summary Service status
// @Description Returns detailed status information including version and uptime
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Detailed status information"
// @Router /api/v1/public/status [get]
func (h *HealthHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "operational",
		"service":   "api-gateway",
		"version":   "1.0.0",
		"uptime":    time.Since(h.startTime).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// AdminUsers returns user administration info (admin only)
func (h *HealthHandler) AdminUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "Admin users endpoint",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// SystemStatus returns system status (admin only)
func (h *HealthHandler) SystemStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "operational",
		"services":  6,
		"uptime":    time.Since(h.startTime).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
