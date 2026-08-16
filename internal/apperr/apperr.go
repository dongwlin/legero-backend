// Package apperr defines transport-agnostic application errors shared by
// the service, infra, and handler layers.
//
// AppError carries a stable machine code, a user-facing message, and an
// optional cause, classified by a Kind. It deliberately knows nothing about
// HTTP: the handler layer (internal/handler/httpresp) maps Kind to HTTP
// status codes and renders the JSON error body.
package apperr

import "errors"

// Kind classifies an application error so transport layers can decide how
// to surface it (for example, which HTTP status code to use).
type Kind string

const (
	// KindInvalidArgument means the caller supplied an invalid request (HTTP 400).
	KindInvalidArgument Kind = "invalid_argument"
	// KindUnauthenticated means the caller is not authenticated (HTTP 401).
	KindUnauthenticated Kind = "unauthenticated"
	// KindForbidden means the caller is authenticated but not allowed (HTTP 403).
	KindForbidden Kind = "forbidden"
	// KindNotFound means the requested resource does not exist (HTTP 404).
	KindNotFound Kind = "not_found"
	// KindConflict means the request conflicts with the current state (HTTP 409).
	KindConflict Kind = "conflict"
	// KindInternal means an unexpected failure occurred (HTTP 500).
	KindInternal Kind = "internal"
)

// AppError is a business/application error. Service and infra layers return
// it directly; handlers translate it into HTTP responses.
type AppError struct {
	Kind    Kind
	Code    string
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return e.Message
}

// Unwrap exposes the underlying cause for errors.Is / errors.As.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New creates an AppError without a cause.
func New(kind Kind, code, message string) *AppError {
	return &AppError{Kind: kind, Code: code, Message: message}
}

// Wrap creates an AppError that wraps a cause.
func Wrap(kind Kind, code, message string, cause error) *AppError {
	return &AppError{Kind: kind, Code: code, Message: message, Cause: cause}
}

// ValidationError reports an invalid request payload or parameter.
func ValidationError(message string) *AppError {
	return New(KindInvalidArgument, "validation_failed", message)
}

// UnauthorizedError reports a missing or invalid credential.
func UnauthorizedError(message string) *AppError {
	return New(KindUnauthenticated, "unauthorized", message)
}

// ForbiddenError reports an authenticated caller lacking permission.
func ForbiddenError(message string) *AppError {
	return New(KindForbidden, "forbidden", message)
}

// NotFoundError reports a missing resource with a stable machine code.
func NotFoundError(code, message string) *AppError {
	return New(KindNotFound, code, message)
}

// ConflictError reports a state conflict with a stable machine code.
func ConflictError(code, message string) *AppError {
	return New(KindConflict, code, message)
}

// InternalError wraps an unexpected failure.
func InternalError(message string, cause error) *AppError {
	return Wrap(KindInternal, "internal_error", message, cause)
}

// Is reports whether err is (or wraps) an AppError of the given kind.
func Is(err error, kind Kind) bool {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Kind == kind
}
