// Package entities holds the user module's request/response DTOs and the
// module-scoped constants (collection and field names) shared by
// repository.go - including its EnsureIndexes - and core.go.
package entities

const (
	CollectionUsers = "users"
)

const (
	FieldID           = "_id"
	FieldName         = "name"
	FieldHandle       = "handle"
	FieldDOB          = "dob"
	FieldPasswordHash = "passwordHash"
	FieldCreatedAt    = "createdAt"
)

const (
	// MaxNameLength is a rune count, not a byte count - see request.go.
	MaxNameLength = 20
	// MinAgeYears is the minimum age to register, checked in UTC so a user
	// whose 18th birthday is today passes.
	MinAgeYears = 18
	// DOBLayout is the only date format the API accepts for dob, so parsing
	// is unambiguous (no locale-dependent day/month ordering).
	DOBLayout = "2006-01-02"
)
