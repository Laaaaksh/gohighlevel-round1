// Package contextkeys defines typed keys for values stored on
// context.Context. A distinct unexported type prevents a bare string key
// from colliding with a key defined by another package sharing the context.
package contextkeys

type contextKey string

const (
	RequestIDKey contextKey = "requestID"
)
