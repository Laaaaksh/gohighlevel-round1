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
	MsgItemNotFound   = "The requested item was not found."
	MsgInvalidRequest = "The request could not be processed."
	MsgInvalidID      = "The provided id is not valid."
	MsgNameRequired   = "Name is required."
	MsgInternalError  = "Something went wrong. Please try again."
)
