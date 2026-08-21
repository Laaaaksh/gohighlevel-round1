package post

// Log messages are static strings; dynamic detail always travels as
// key-value pairs on the same call - see go-coding-standards.
const (
	logMsgPostCreated          = "post created"
	logMsgCreatePostFailed     = "failed to create post"
	logMsgListPostsFailed      = "failed to list posts"
	logMsgGeneratePostIDFailed = "failed to generate post id"
	logMsgBindRequestFailed    = "failed to bind post request"

	logFieldError  = "error"
	logFieldPostID = "postID"
	logFieldUserID = "userID"
)
