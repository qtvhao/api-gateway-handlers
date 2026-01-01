// Package handlers provides HTTP request handlers for the API Gateway.
//
// Associated Frontend Files:
//   - web/app/src/components/layout/Sidebar.tsx (navigation menu)
//   - web/app/src/components/layout/AppLayout.tsx (app layout with navigation)
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NavigationHandler handles navigation configuration
type NavigationHandler struct {
	logger *zap.Logger
}

// NewNavigationHandler creates a new NavigationHandler
func NewNavigationHandler(logger *zap.Logger) *NavigationHandler {
	return &NavigationHandler{
		logger: logger,
	}
}

// NavItem represents a navigation item
// Note: Field names match frontend expectations in Sidebar.tsx
type NavItem struct {
	Name string `json:"name"`
	Href string `json:"href"`
	Icon string `json:"icon"`
}

// GetNavigation returns the navigation configuration
func (h *NavigationHandler) GetNavigation(c *gin.Context) {
	items := []NavItem{
		{
			Name: "Dashboard",
			Icon: "LayoutDashboard",
			Href: "/",
		},
		{
			Name: "Employees",
			Icon: "Users",
			Href: "/employees",
		},
		{
			Name: "Skills",
			Icon: "Award",
			Href: "/skills",
		},
		{
			Name: "Assignments",
			Icon: "GitBranch",
			Href: "/assignments",
		},
		{
			Name: "Projects",
			Icon: "FolderKanban",
			Href: "/projects",
		},
		{
			Name: "Settings",
			Icon: "Settings",
			Href: "/settings",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
	})
}
