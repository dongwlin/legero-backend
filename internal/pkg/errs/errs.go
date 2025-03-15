package errs

import "errors"

type Error struct {
	StatusCode int
	Message    string
	Data       map[string]any
}

func (e *Error) Error() string {
	return e.Message
}

var (
	// user
	ErrUserNotFound             = errors.New("user not found")
	ErrUsernameAlreadyExists    = errors.New("username already exists")
	ErrPhoneNumberAlreadyExists = errors.New("phone number already exists")

	// auth
	ErrWrongPassword = errors.New("incorrect password")
	ErrUserBlocked   = errors.New("user is blocked")
	ErrInvalidToken  = errors.New("invalid token")

	// order item
	ErrInvalidCustomNoodleType = errors.New("invalid custom noodle type")
)
