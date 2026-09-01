package httpresp

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/apperr"
)

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

// AbortError aborts the request with the JSON error body for err, wrapped in
// the unified Response envelope so success and error responses share one
// contract: the envelope code carries the machine-readable business status and
// data is null.
//
// *apperr.AppError values are mapped through their Kind to an HTTP status
// (400/401/403/404/409/500). Any other error becomes a generic
// "internal_error" 500 response, so wrap expected failures explicitly.
func AbortError(c *gin.Context, err error) {
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		c.AbortWithStatusJSON(statusForKind(appErr.Kind), Response{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
		Code:    "internal_error",
		Message: http.StatusText(http.StatusInternalServerError),
	})
}
