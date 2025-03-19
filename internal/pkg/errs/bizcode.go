package errs

const (
	BizCodeInvalidParams = 100001

	// user
	BizCodeUserNotFound             = 200001
	BizCodeUsernameAlreadyExists    = 200002
	BizCodePhoneNumberAlreadyExists = 200003

	// auth
	BizCodeIncorrectPassword = 300001
	BizCodeUserBlocked       = 300002
	BizCodeInvalidToken      = 300003

	// order item
	BizCodeInvalidCustomNoodleType = 400001
)
