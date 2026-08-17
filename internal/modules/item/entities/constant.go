// Package entities holds the item module's request/response DTOs and the
// module-scoped constants (collection and field names) shared by
// repository.go - including its EnsureIndexes - and core.go.
package entities

const (
	CollectionItems = "items"
)

const (
	FieldID          = "_id"
	FieldName        = "name"
	FieldDescription = "description"
	FieldCreatedAt   = "createdAt"
	FieldUpdatedAt   = "updatedAt"
)

const (
	MaxNameLength        = 200
	MaxDescriptionLength = 2000
)
