// Package entities holds the follow module's module-scoped constants
// (collection and field names), shared by repository.go - including its
// EnsureIndexes - and core.go. Follow has no request/response body: the
// two ids travel as a path param and a query param (see server.go).
package entities

const (
	CollectionFollows = "follows"
)

const (
	FieldID         = "_id"
	FieldFollowerID = "followerId"
	FieldFolloweeID = "followeeId"
	FieldCreatedAt  = "createdAt"
)
