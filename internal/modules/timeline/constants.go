package timeline

// includeOwnPosts is the §2.6.1 decision: the brief's strict reading (only
// people you follow) would say no, but most real timeline products fold
// the viewer's own posts in, and that is what users expect from a feed
// named "timeline". A single named constant, not a scattered literal,
// documents the choice and makes it a one-line change to reverse.
const includeOwnPosts = true

// Log messages are static strings; dynamic detail always travels as
// key-value pairs on the same call - see go-coding-standards.
const (
	logMsgListFolloweesFailed = "failed to list followees for timeline"
	logMsgBindQueryFailed     = "failed to bind timeline query"

	logFieldError  = "error"
	logFieldUserID = "userID"
)
