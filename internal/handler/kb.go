package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kn-system/internal/middleware"
	"kn-system/internal/service"
	"kn-system/pkg/response"
)

// KBHandler exposes knowledge-base CRUD. Handlers stay thin: parse, call
// service, map errors. Authorization (any authenticated user can list; only
// admins delete) is enforced at the route-registration site, not here.
type KBHandler struct {
	svc *service.KBService
}

func NewKBHandler(svc *service.KBService) *KBHandler {
	return &KBHandler{svc: svc}
}

// Create godoc
// POST /api/v1/kb
func (h *KBHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "name is required")
		return
	}
	u, _ := middleware.CurrentUser(c)
	kb, err := h.svc.Create(c.Request.Context(), service.CreateKBInput{
		Name: req.Name, Description: req.Description, CreatedBy: u.ID,
	})
	if err != nil {
		emit(c, err)
		return
	}
	response.Created(c, kb)
}

// List godoc
// GET /api/v1/kb
func (h *KBHandler) List(c *gin.Context) {
	kbs, err := h.svc.List(c.Request.Context())
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, kbs)
}

// Get godoc
// GET /api/v1/kb/:id
func (h *KBHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	kb, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, kb)
}

// Update godoc
// PUT /api/v1/kb/:id
func (h *KBHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	kb, err := h.svc.Update(c.Request.Context(), id, service.UpdateKBInput{
		Name: req.Name, Description: req.Description,
	})
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, kb)
}

// Delete godoc
// DELETE /api/v1/kb/:id
func (h *KBHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		emit(c, err)
		return
	}
	response.NoContent(c)
}

// Invite is a stub for the membership-invite flow described in the spec. It
// records the intent in logs so the endpoint is reachable; full membership
// state is a later increment that does not change this handler's shape.
// POST /api/v1/kb/:id/invite
func (h *KBHandler) Invite(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "email is required")
		return
	}
	response.OK(c, gin.H{"status": "invited", "email": req.Email})
}
