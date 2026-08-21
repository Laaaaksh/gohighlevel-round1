package follow

// Log messages are static strings; dynamic detail always travels as
// key-value pairs on the same call - see go-coding-standards.
const (
	logMsgFollowCreated       = "follow edge created"
	logMsgFollowFailed        = "failed to create follow edge"
	logMsgListFolloweesFailed = "failed to list followees"

	logFieldError      = "error"
	logFieldFollowerID = "followerID"
	logFieldFolloweeID = "followeeID"
)
