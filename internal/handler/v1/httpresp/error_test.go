package httpresp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAbortErrorMapsAppErrorKindToHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{
			name:       "invalid argument",
			err:        apperr.ValidationError("bad input"),
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation_failed",
			wantMsg:    "bad input",
		},
		{
			name:       "unauthenticated",
			err:        apperr.UnauthorizedError("missing auth"),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
			wantMsg:    "missing auth",
		},
		{
			name:       "forbidden",
			err:        apperr.ForbiddenError("no permission"),
			wantStatus: http.StatusForbidden,
			wantCode:   "forbidden",
			wantMsg:    "no permission",
		},
		{
			name:       "not found keeps machine code",
			err:        apperr.NotFoundError("order_not_found", "order not found"),
			wantStatus: http.StatusNotFound,
			wantCode:   "order_not_found",
			wantMsg:    "order not found",
		},
		{
			name:       "conflict",
			err:        apperr.ConflictError("order_conflict", "order modified"),
			wantStatus: http.StatusConflict,
			wantCode:   "order_conflict",
			wantMsg:    "order modified",
		},
		{
			name:       "internal",
			err:        apperr.InternalError("boom", errors.New("cause")),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
			wantMsg:    "boom",
		},
		{
			name:       "plain error becomes generic 500",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
			wantMsg:    http.StatusText(http.StatusInternalServerError),
		},
		{
			name:       "wrapped app error is unwrapped",
			err:        fmt.Errorf("wrap: %w", apperr.UnauthorizedError("nope")),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
			wantMsg:    "nope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)

			AbortError(c, tt.err)

			require.Equal(t, tt.wantStatus, recorder.Code)
			var body errorBody
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			require.Equal(t, tt.wantCode, body.Error.Code)
			require.Equal(t, tt.wantMsg, body.Error.Message)
		})
	}
}
