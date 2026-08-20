package handler

import (
	"github.com/gin-gonic/gin"

	"kn-system/internal/middleware"
	"kn-system/internal/model"
	"kn-system/internal/service"
	"kn-system/pkg/response"
)

// AuthHandler translates HTTP requests into AuthService calls. It is the only
// handler concerned with credentials, keeping registration/login endpoints
// isolated from the rest of the API surface.
type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register godoc
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	user, token, err := h.svc.Register(c.Request.Context(), service.RegisterInput{
		Email: req.Email, Password: req.Password, Name: req.Name,
		Role: model.RoleMember,
	})
	if err != nil {
		emit(c, err)
		return
	}
	response.Created(c, gin.H{
		"user":  user,
		"token": token,
	})
}

// Login godoc
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	user, token, err := h.svc.Login(c.Request.Context(), service.LoginInput{
		Email: req.Email, Password: req.Password,
	})
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, gin.H{
		"user":  user,
		"token": token,
	})
}

// Me returns the currently authenticated user's profile.
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	u, ok := middleware.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "not authenticated")
		return
	}
	response.OK(c, u)
}
