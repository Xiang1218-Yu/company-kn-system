package errors

import (
	"errors"
	"net/http"
)

// Kind classifies an error so handlers can map it to an HTTP status without
// importing the concrete type. This keeps the service layer free of HTTP
// concerns and lets handlers stay thin.
type Kind int

const (
	KindUnknown Kind = iota
	KindNotFound
	KindValidation
	KindUnauthorized
	KindForbidden
	KindConflict
	KindInternal
)

// Error is the application error type. Services return *Error to carry both a
// user-facing message and a classification. Anything not *Error is treated as
// KindInternal.
type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

// New constructors make constructing classified errors concise at call sites.
func New(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, cause error) *Error {
	return &Error{Kind: kind, Message: message, Cause: cause}
}

// Classify inspects an error and reports its Kind, defaulting to Internal for
// non-application errors.
func Classify(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}

// MessageOf returns the error text, or a generic fallback so internal details
// are never leaked to the client.
func MessageOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Message
	}
	return "internal error"
}

// StatusFor maps a Kind to an HTTP status code. Centralizing this mapping keeps
// the handler-to-status contract in one place.
func StatusFor(k Kind) int {
	switch k {
	case KindNotFound:
		return http.StatusNotFound
	case KindValidation:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// CodeFor returns the stable error code string the frontend switches on.
func CodeFor(k Kind) string {
	switch k {
	case KindNotFound:
		return "not_found"
	case KindValidation:
		return "bad_request"
	case KindUnauthorized:
		return "unauthorized"
	case KindForbidden:
		return "forbidden"
	case KindConflict:
		return "conflict"
	default:
		return "internal_error"
	}
}
