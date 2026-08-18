package apperror

// Field keys attached to an error via WithField, identifying which request
// field the error relates to.
const (
	FieldID          = "id"
	FieldName        = "name"
	FieldDescription = "description"
	FieldHandle      = "handle"
	FieldDOB         = "dob"
	FieldPassword    = "password"
	FieldUserID      = "userId"
	FieldTitle       = "title"
	FieldBody        = "body"
	FieldCursor      = "cursor"
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
	MsgUserNotFound     = "The requested user was not found."
)

// Per-field validation details. These travel in the Fields map keyed by the
// field they describe, so a client can point at the input that was rejected
// instead of parsing the summary message. Fields carries these fixed messages
// only, never the submitted value - callers render them straight into the UI,
// so echoing input back would put unvalidated text on the page.
const (
	MsgNameRequired       = "Name is required."
	MsgNameTooLong        = "Name is longer than the maximum allowed length."
	MsgDescriptionTooLong = "Description is longer than the maximum allowed length."
	MsgIDMalformed        = "This id is not in the expected format."
	MsgIDUnknown          = "No item exists with this id."

	MsgHandleRequired   = "Handle is required."
	MsgHandleTaken      = "This handle is already taken."
	MsgDOBRequired      = "Date of birth is required."
	MsgDOBMalformed     = "Date of birth is not a valid date."
	MsgDOBTooYoung      = "You must be at least 18 years old to register."
	MsgPasswordRequired = "Password is required."

	MsgUserIDRequired  = "userId is required."
	MsgUserIDMalformed = "userId is not a valid identifier."
	MsgUserIDUnknown   = "No user exists with this id."

	MsgTitleRequired = "Title is required."
	MsgTitleTooLong  = "Title is longer than the maximum allowed length."
	MsgBodyRequired  = "Body is required."
	MsgBodyTooLong   = "Body is longer than the maximum allowed length."

	MsgSelfFollowNotAllowed = "A user cannot follow themselves."
	MsgFolloweeNotFound     = "The user to follow was not found."
	MsgFollowerNotFound     = "The following user was not found."

	MsgCursorMalformed = "The pagination cursor is not valid."
)
