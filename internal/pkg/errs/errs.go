package errs

import "errors"

var (
	// user
	ErrUserNotFound             = errors.New("user not found")
	ErrUsernameAlreadyExists    = errors.New("username already exists")
	ErrPhoneNumberAlreadyExists = errors.New("phone number already exists")

	// auth
	ErrWrongPassword = errors.New("incorrect password")
	ErrUserBlocked   = errors.New("user is blocked")
	ErrInvalidToken  = errors.New("invalid token")
)
