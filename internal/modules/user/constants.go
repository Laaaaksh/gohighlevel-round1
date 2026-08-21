package user

// Log messages are static strings; dynamic detail always travels as
// key-value pairs on the same call - see go-coding-standards.
const (
	logMsgUserRegistered       = "user registered"
	logMsgRegisterUserFailed   = "failed to register user"
	logMsgHashPasswordFailed   = "failed to hash password"
	logMsgLookupUserFailed     = "failed to look up user"
	logMsgBindRequestFailed    = "failed to bind register user request"
	logMsgGenerateUserIDFailed = "failed to generate user id"

	logFieldError  = "error"
	logFieldUserID = "userID"
)
