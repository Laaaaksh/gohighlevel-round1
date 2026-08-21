// Package entities holds the post module's request/response DTOs and the
// module-scoped constants (collection and field names) shared by
// repository.go - including its EnsureIndexes - and core.go.
package entities

const (
	CollectionPosts = "posts"
)

const (
	FieldID        = "_id"
	FieldTitle     = "title"
	FieldBody      = "body"
	FieldUserID    = "userId"
	FieldCreatedAt = "createdAt"
)

const (
	// MaxTitleLength and MaxBodyLength are rune counts, not byte counts -
	// see request.go.
	MaxTitleLength = 20
	MaxBodyLength  = 300

	// DefaultPageSize and MaxPageSize bound every list endpoint. A limit
	// above MaxPageSize is clamped, not rejected - see the project report's
	// §2.6.3 decision.
	DefaultPageSize = 20
	MaxPageSize     = 100
)
