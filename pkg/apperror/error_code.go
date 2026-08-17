// Package apperror defines the service's error wire format: a stable error
// code, an HTTP status derived from it, and a public message that never
// leaks internal (driver, database) detail to the client.
package apperror

import "net/http"

// Code is a stable, machine-readable error identifier. Clients can branch on
// it without parsing message text.
type Code string

const (
	CodeBadRequest      Code = "BAD_REQUEST"
	CodeNotFound        Code = "NOT_FOUND"
	CodeConflict        Code = "CONFLICT"
	CodeValidationError Code = "VALIDATION_ERROR"
	CodeInternalError   Code = "INTERNAL_ERROR"
)

// codeToStatus is the single place that maps an error code to an HTTP
// status, so server.go files never hardcode a status number.
var codeToStatus = map[Code]int{
	CodeBadRequest:      http.StatusBadRequest,
	CodeNotFound:        http.StatusNotFound,
	CodeConflict:        http.StatusConflict,
	CodeValidationError: http.StatusBadRequest,
	CodeInternalError:   http.StatusInternalServerError,
}

// HTTPStatus returns the HTTP status for the code, defaulting to 500 for an
// unmapped code rather than panicking or returning a zero value.
func (c Code) HTTPStatus() int {
	if status, ok := codeToStatus[c]; ok {
		return status
	}
	return http.StatusInternalServerError
}
