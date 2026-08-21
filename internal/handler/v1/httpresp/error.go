package httpresp

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/apperr"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// statusForKind maps an apperr.Kind to the HTTP status code used for it.
func statusForKind(kind apperr.Kind) int {
	switch kind {
	case apperr.KindInvalidArgument:
		return http.StatusBadRequest
	case apperr.KindUnauthenticated:
		return http.StatusUnauthorized
	case apperr.KindForbidden:
		return http.StatusForbidden
	case apperr.KindNotFound:
		return http.StatusNotFound
	case apperr.KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// AbortError aborts the request with the JSON error body for err.
//
// *apperr.AppError values are mapped through their Kind to an HTTP status
// (400/401/403/404/409/500). Any other error becomes a generic
// "internal_error" 500 response, so wrap expected failures explicitly.
func AbortError(c *gin.Context, err error) {
	var body errorBody
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		body.Error.Code = appErr.Code
		body.Error.Message = appErr.Message
		c.AbortWithStatusJSON(statusForKind(appErr.Kind), body)
		return
	}

	body.Error.Code = "internal_error"
	body.Error.Message = http.StatusText(http.StatusInternalServerError)
	c.AbortWithStatusJSON(http.StatusInternalServerError, body)
}
