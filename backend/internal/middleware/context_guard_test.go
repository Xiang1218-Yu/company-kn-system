package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"kn-system/internal/handler"
	"kn-system/internal/middleware"
	"kn-system/internal/model"
)

func TestContextGuardsRejectMalformedOrMissingIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("malformed role", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.CtxRole, "admin")
		})
		router.GET("/protected", middleware.RequireRole(model.RoleAdmin), func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
		rec := httptest.NewRecorder()
		panicked := false
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					panicked = true
					t.Errorf("RequireRole panicked for malformed role: %v", recovered)
				}
			}()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/protected", nil))
		}()
		if !panicked && rec.Code != http.StatusUnauthorized {
			t.Errorf("RequireRole returned HTTP %d for malformed role, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		router := gin.New()
		dashboard := handler.NewDashboardHandler(nil)
		// GET /api/v1/dashboard
		router.GET("/api/v1/dashboard", dashboard.Load)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("dashboard returned HTTP %d without authenticated user, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}
