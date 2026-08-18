package entities

// Struct tags must be literals, so MaxNameLength cannot be referenced
// directly in the tag below. It is a first pass only: core.go re-checks
// every write against that constant, so a drifted tag cannot let a bad
// value reach the database. Both layers count runes, not bytes.

// RegisterUserRequest is the inbound shape for POST /users. dob is a plain
// string here (not time.Time) because Gin's JSON binding would otherwise
// accept any RFC 3339 timestamp; core.go parses it against DOBLayout so a
// malformed date is rejected with the same VALIDATION_ERROR shape as every
// other field, not a binding-layer 400 with no field detail.
type RegisterUserRequest struct {
	Name     string `json:"name" binding:"required,max=20"`
	Handle   string `json:"handle" binding:"required"`
	DOB      string `json:"dob" binding:"required"`
	Password string `json:"password" binding:"required"`
}
