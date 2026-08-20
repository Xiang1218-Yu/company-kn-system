package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError carries a machine-readable error code alongside a human message so
// the frontend can branch on known failure modes (validation, auth, not-found)
// without parsing strings.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Body is the single envelope every endpoint returns. Data holds the payload
// on success; Error holds structured details on failure. Keeping one envelope
// means the frontend has exactly one shape to parse.
type Body struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Success: true, Data: data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Body{Success: true, Data: data})
}

func NoContent(c *gin.Context) {
	c.JSON(http.StatusOK, Body{Success: true, Data: nil})
}

// Fail writes an error envelope with the given HTTP status. The code string is
// the stable contract the frontend switches on; the message is for display.
func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, Body{Success: false, Error: &APIError{Code: code, Message: message}})
}

// Convenience constructors for the common HTTP error buckets. Centralizing them
// here keeps handlers from inventing ad-hoc codes and statuses.
func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, "bad_request", message)
}

func Unauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, "unauthorized", message)
}

func Forbidden(c *gin.Context, message string) {
	Fail(c, http.StatusForbidden, "forbidden", message)
}

func NotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, "not_found", message)
}

func Internal(c *gin.Context, message string) {
	Fail(c, http.StatusInternalServerError, "internal_error", message)
}
