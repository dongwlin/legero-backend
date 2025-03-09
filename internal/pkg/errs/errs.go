package errs

import "errors"

var (
	// user
	ErrUserNotFound = errors.New("user not found")

	// auth
	ErrWrongPassword = errors.New("incorrect password")
)
