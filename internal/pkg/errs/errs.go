package errs

import (
	"fmt"
	"net/http"
)

type Error struct {
	bizCode  int
	httpCode int
	message  string
	details  map[string]any
	internal error
}

func (e *Error) Error() string {

	if e.internal != nil {
		return fmt.Sprintf("%s: %v", e.message, e.internal)
	}

	return e.message
}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && e.bizCode == t.BizCode()
}

func (e *Error) Unwrap() error {
	return e.internal
}

func (e *Error) BizCode() int {
	return e.bizCode
}

func (e *Error) HTTPCode() int {
	return e.httpCode
}

func (e *Error) Message() string {
	return e.message
}

func (e *Error) Details() map[string]any {
	return e.details
}

func (e *Error) Wrap(err error) *Error {
	return &Error{
		bizCode:  e.bizCode,
		httpCode: e.httpCode,
		message:  e.message,
		details:  e.details,
		internal: err,
	}
}

func (e *Error) WithDetails(details map[string]any) *Error {

	if details == nil {
		details = make(map[string]any)
	}

	return &Error{
		bizCode:  e.bizCode,
		httpCode: e.httpCode,
		message:  e.message,
		details:  details,
		internal: e.internal,
	}
}

func New(httpCode, bizCode int, message string, internal error) *Error {
	return &Error{
		bizCode:  bizCode,
		httpCode: httpCode,
		message:  message,
		details:  make(map[string]any),
		internal: internal,
	}
}

var (
	ErrInvalidParams = New(http.StatusBadRequest, BizCodeInvalidParams, "invalid params", nil)

	// user
	ErrUserNotFound             = New(http.StatusBadRequest, BizCodeUserNotFound, "user not found", nil)
	ErrUsernameAlreadyExists    = New(http.StatusBadRequest, BizCodeUsernameAlreadyExists, "username already exists", nil)
	ErrPhoneNumberAlreadyExists = New(http.StatusBadRequest, BizCodePhoneNumberAlreadyExists, "phone number already exists", nil)

	// auth
	ErrWrongPassword = New(http.StatusBadRequest, BizCodeIncorrectPassword, "incorrect password", nil)
	ErrUserBlocked   = New(http.StatusBadRequest, BizCodeUserBlocked, "user is blocked", nil)
	ErrInvalidToken  = New(http.StatusBadRequest, BizCodeInvalidToken, "invalid token", nil)

	// order item
	ErrInvalidCustomNoodleType = New(http.StatusBadRequest, BizCodeInvalidCustomNoodleType, "invalid custom noodle type", nil)
)
