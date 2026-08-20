package handler

import (
	"github.com/gin-gonic/gin"

	"kn-system/internal/middleware"
	"kn-system/internal/model"
	"kn-system/internal/service"
	"kn-system/pkg/response"
)

// DashboardHandler exposes operational metrics. Restricted to managers/admins
// at route registration since it surfaces aggregate usage data.
type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// Load godoc
// GET /api/v1/dashboard
func (h *DashboardHandler) Load(c *gin.Context) {
	// Enforce manager-or-above here too, as defence in depth alongside the
	// route-level RequireRole.
	u, ok := middleware.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "not authenticated")
		return
	}
	if u.Role != model.RoleAdmin && u.Role != model.RoleManager {
		response.Forbidden(c, "insufficient permissions")
		return
	}
	d, err := h.svc.Load(c.Request.Context())
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, d)
}
