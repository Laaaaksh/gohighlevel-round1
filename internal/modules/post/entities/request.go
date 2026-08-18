package entities

// Struct tags must be literals, so MaxTitleLength/MaxBodyLength cannot be
// referenced directly in the tags below. They are a first pass only:
// core.go re-checks every write against those constants, so a drifted tag
// cannot let a bad value reach the database. Both layers count runes, not
// bytes.

// CreatePostRequest is the inbound shape for POST /posts.
type CreatePostRequest struct {
	Title  string `json:"title" binding:"required,max=20"`
	Body   string `json:"body" binding:"required,max=300"`
	UserID string `json:"userId" binding:"required"`
}

// ListPostsQuery is the inbound shape for GET /posts's query string.
// Cursor and Limit are optional: an empty cursor means "first page", and a
// zero/absent limit falls back to DefaultPageSize.
type ListPostsQuery struct {
	UserID string `form:"userId"`
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}
