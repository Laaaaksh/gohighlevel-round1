package apperror

// Field keys attached to an error via WithField, identifying which request
// field the error relates to.
const (
	FieldID          = "id"
	FieldName        = "name"
	FieldDescription = "description"
)

// User-facing messages. These are shown to API clients, so they must never
// contain driver, schema, or stack detail - see error.go for how internal
// detail is kept separate.
const (
	MsgItemNotFound     = "The requested item was not found."
	MsgInvalidRequest   = "The request could not be processed."
	MsgInvalidID        = "The provided id is not valid."
	MsgValidationFailed = "The request contains an invalid field."
	MsgInternalError    = "Something went wrong. Please try again."
)

// Per-field validation details. These travel in the Fields map keyed by the
// field they describe, so a client can point at the input that was rejected
// instead of parsing the summary message.
const (
	MsgNameRequired       = "Name is required."
	MsgNameTooLong        = "Name is longer than the maximum allowed length."
	MsgDescriptionTooLong = "Description is longer than the maximum allowed length."
)
