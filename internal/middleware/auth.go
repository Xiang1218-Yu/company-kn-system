package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	apperr "kn-system/internal/errors"
	"kn-system/internal/model"
	"kn-system/internal/service"
	"kn-system/pkg/response"
)

// Context keys used to stash the authenticated user id/role. Typed string consts
// avoid collisions and make lookup sites self-documenting.
const (
	CtxUserID = "ctx.userID"
	CtxRole   = "ctx.role"
	CtxUser   = "ctx.user"
)

// Auth returns middleware that validates the Bearer token and loads the user
// into context. Requests without a valid token are rejected here; optional
// auth (for public-but-personalized routes) would use a separate middleware.
func Auth(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			response.Unauthorized(c, "missing token")
			c.Abort()
			return
		}
		user, err := svc.Authenticate(c.Request.Context(), token)
		if err != nil {
			kind := apperr.Classify(err)
			response.Fail(c, apperr.StatusFor(kind), apperr.CodeFor(kind), apperr.MessageOf(err))
			c.Abort()
			return
		}
		c.Set(CtxUser, user)
		c.Set(CtxUserID, user.ID)
		c.Set(CtxRole, user.Role)
		c.Next()
	}
}

// RequireRole gates a route to the listed roles. Must run after Auth so the
// role is in context. An empty allowed list means "any authenticated user".
func RequireRole(allowed ...model.Role) gin.HandlerFunc {
	set := make(map[model.Role]bool, len(allowed))
	for _, r := range allowed {
		set[r] = true
	}
	return func(c *gin.Context) {
		role, ok := c.Get(CtxRole)
		if !ok {
			response.Unauthorized(c, "not authenticated")
			c.Abort()
			return
		}
		if len(set) == 0 {
			c.Next()
			return
		}
		if !set[role.(model.Role)] {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

// CurrentUser retrieves the authenticated user stashed by Auth. Returns the
// zero value (and false) when no user is present, so callers can fail closed.
func CurrentUser(c *gin.Context) (model.User, bool) {
	v, ok := c.Get(CtxUser)
	if !ok {
		return model.User{}, false
	}
	u, _ := v.(model.User)
	return u, true
}

// bearerToken extracts the token from an Authorization header, tolerating
// missing/wrong schemes by returning "" (which the caller rejects uniformly).
func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
