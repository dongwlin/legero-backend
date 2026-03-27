package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewError(status int, code, message string) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

func WrapError(status int, code, message string, err error) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func ValidationError(message string) *AppError {
	return NewError(http.StatusBadRequest, "validation_failed", message)
}

func UnauthorizedError(message string) *AppError {
	return NewError(http.StatusUnauthorized, "unauthorized", message)
}

func ForbiddenError(message string) *AppError {
	return NewError(http.StatusForbidden, "forbidden", message)
}

func NotFoundError(code, message string) *AppError {
	return NewError(http.StatusNotFound, code, message)
}

func ConflictError(code, message string) *AppError {
	return NewError(http.StatusConflict, code, message)
}

func InternalError(message string, err error) *AppError {
	return WrapError(http.StatusInternalServerError, "internal_error", message, err)
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func AbortError(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		var body errorBody
		body.Error.Code = appErr.Code
		body.Error.Message = appErr.Message
		c.AbortWithStatusJSON(appErr.Status, body)
		return
	}

	var body errorBody
	body.Error.Code = "internal_error"
	body.Error.Message = http.StatusText(http.StatusInternalServerError)
	c.AbortWithStatusJSON(http.StatusInternalServerError, body)
}
