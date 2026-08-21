package entities

import "time"

// PostResponse is the outbound shape for a single post, used by both
// "my posts" and the timeline - the brief specifies the same shape for
// both. It deliberately omits the author's handle: see the project report's
// §2.6.2 decision and how to add it later (denormalize at write time)
// without an N+1 read.
type PostResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	UserID    string    `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreatePostResponse is the outbound shape for POST /posts. The brief's
// literal contract is a bare postId string; every other response in this
// service is a JSON object, so this wraps it the same way - see the
// project report for this deliberate deviation.
type CreatePostResponse struct {
	PostID string `json:"postId"`
}

// HeaderNextCursor carries the cursor for the next page. The brief's literal
// list-endpoint contract is a bare JSON array, so the cursor cannot live in
// the body without breaking that shape - it travels as a response header
// instead, present only when a further page exists.
const HeaderNextCursor = "X-Next-Cursor"
