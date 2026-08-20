package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kn-system/internal/auth"
	"kn-system/internal/config"
	"kn-system/internal/model"
	"kn-system/internal/repository"
	"kn-system/internal/service"
)

func TestRegisterPublicRoleCannotCreatePrivilegedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:register_public_role?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}

	jwt := auth.NewJWT(config.JWTConfig{Secret: "test-secret", Issuer: "test", ExpireHours: 1})
	svc := service.NewAuthService(repository.NewUserRepo(db), auth.NewPassword(), jwt)
	router := gin.New()
	handler := NewAuthHandler(svc)
	// POST /api/v1/auth/register
	router.POST("/api/v1/auth/register", handler.Register)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"new@example.test","password":"password1","name":"new user","role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register returned HTTP %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		Data struct {
			User  model.User `json:"user"`
			Token string     `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if body.Data.User.Role != model.RoleMember {
		t.Fatalf("register returned elevated role %q for a public request", body.Data.User.Role)
	}
	claims, err := jwt.Verify(body.Data.Token)
	if err != nil {
		t.Fatalf("verify registration token: %v", err)
	}
	if claims.Role != model.RoleMember {
		t.Fatalf("registration token preserved elevated role %q", claims.Role)
	}
}
