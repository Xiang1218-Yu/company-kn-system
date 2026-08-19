package handler

import (
	"github.com/gin-gonic/gin"

	apperr "kn-system/internal/errors"
	"kn-system/pkg/response"
)

// emit maps a service-layer error to the HTTP response envelope. Centralizing
// the status/code mapping here keeps every handler focused on request parsing
// and response shaping.
func emit(c *gin.Context, err error) {
	kind := apperr.Classify(err)
	response.Fail(c, apperr.StatusFor(kind), apperr.CodeFor(kind), apperr.MessageOf(err))
}
