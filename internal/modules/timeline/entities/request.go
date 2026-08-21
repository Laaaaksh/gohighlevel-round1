// Package entities holds the timeline module's request DTO. The response
// shape is identical to the post module's (§2.5 specifies the same post
// shape as §2.3), so timeline reuses post/entities.PostResponse directly
// instead of duplicating it.
package entities

// TimelineQuery is the inbound shape for GET /timeline's query string.
// Cursor and Limit are optional: an empty cursor means "first page", and a
// zero/absent limit falls back to the post module's DefaultPageSize.
type TimelineQuery struct {
	UserID string `form:"userId"`
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}
