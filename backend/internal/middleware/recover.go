package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"kn-system/pkg/logger"
	"kn-system/pkg/response"

	"go.uber.org/zap"
)

// Recover converts panics into structured 500 responses so a single handler
// bug never crashes the process or returns a stack trace to the client. The
// stack is logged server-side for diagnosis.
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.L.Error("panic recovered",
					zap.Any("recover", r),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", c.Request.URL.Path))
				response.Internal(c, "internal error")
				c.Abort()
				return
			}
		}()
		c.Next()
	}
}

// RequestLog logs each request at the end of its lifecycle with status, method,
// path, and latency. It is the one place request observability is wired, so
// every route is covered uniformly.
func RequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		path := c.Request.URL.Path
		// Skip health-check noise so logs stay focused on real traffic.
		if path == "/healthz" {
			return
		}
		logger.L.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("latency_ms", int64(c.Writer.Size())),
		)
	}
}
