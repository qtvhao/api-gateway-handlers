// Package handlers provides HTTP request handlers for the API Gateway.
//
// Associated Frontend Files:
//   - web/app/src/pages/settings/NotificationSettings.tsx (notification settings UI)
//   - web/app/src/config/notificationConfig.ts (notification API client)
package handlers

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SettingsHandler handles user settings
type SettingsHandler struct {
	logger              *zap.Logger
	userPreferences     map[string]NotificationPreferences
	userPreferencesMu   sync.RWMutex
}

// NewSettingsHandler creates a new SettingsHandler
func NewSettingsHandler(logger *zap.Logger) *SettingsHandler {
	return &SettingsHandler{
		logger:          logger,
		userPreferences: make(map[string]NotificationPreferences),
	}
}

// NotificationItem represents a notification setting item
type NotificationItem struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// NotificationPreferences represents user notification preferences
type NotificationPreferences map[string]bool

// NotificationSettingsResponse represents the response for notification settings
type NotificationSettingsResponse struct {
	Items       []NotificationItem      `json:"items"`
	Preferences NotificationPreferences `json:"preferences"`
}

// GetNotificationSettings returns notification settings for the authenticated user
func (h *SettingsHandler) GetNotificationSettings(c *gin.Context) {
	// Get user ID from JWT claims (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	// Default notification items
	items := []NotificationItem{
		{
			Key:         "email",
			Label:       "Email Notifications",
			Description: "Receive notifications via email",
		},
		{
			Key:         "push",
			Label:       "Push Notifications",
			Description: "Receive browser push notifications",
		},
		{
			Key:         "assignments",
			Label:       "Assignment Updates",
			Description: "Get notified when assignments change",
		},
		{
			Key:         "skillUpdates",
			Label:       "Skill Updates",
			Description: "Get notified about skill taxonomy changes",
		},
	}

	// Get user preferences or return defaults
	h.userPreferencesMu.RLock()
	prefs, ok := h.userPreferences[userID.(string)]
	h.userPreferencesMu.RUnlock()

	if !ok {
		// Default preferences
		prefs = NotificationPreferences{
			"email":        false,
			"push":         true,
			"assignments":  false,
			"skillUpdates": true,
		}
	}

	c.JSON(http.StatusOK, NotificationSettingsResponse{
		Items:       items,
		Preferences: prefs,
	})
}

// UpdateNotificationSettings updates notification preferences for the authenticated user
func (h *SettingsHandler) UpdateNotificationSettings(c *gin.Context) {
	// Get user ID from JWT claims
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	var request struct {
		Preferences NotificationPreferences `json:"preferences"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	// Store preferences
	h.userPreferencesMu.Lock()
	h.userPreferences[userID.(string)] = request.Preferences
	h.userPreferencesMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"preferences": request.Preferences,
	})
}
