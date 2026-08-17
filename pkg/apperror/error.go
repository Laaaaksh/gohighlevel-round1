package apperror

import "fmt"

// Error is the service's error type. Message is safe to return to a client;
// internal holds the underlying cause (e.g. a raw driver error) for logging
// only and is never serialized.
type Error struct {
	Code     Code
	Message  string
	Fields   map[string]string
	internal error
}

// New creates a client-facing error with no internal cause.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap creates a client-facing error that also carries an internal cause,
// so the caller can log the real driver/database error without exposing it.
func Wrap(code Code, message string, internal error) *Error {
	return &Error{Code: code, Message: message, internal: internal}
}

func (e *Error) Error() string {
	if e.internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.internal)
	}
	return e.Message
}

// Unwrap exposes the internal cause to errors.Is/errors.As, while Response
// (see response.go) keeps it out of what gets sent to the client.
func (e *Error) Unwrap() error {
	return e.internal
}

// WithField attaches a field key/value pair to the error and returns it for
// chaining at the call site.
func (e *Error) WithField(key, value string) *Error {
	if e.Fields == nil {
		e.Fields = make(map[string]string)
	}
	e.Fields[key] = value
	return e
}
