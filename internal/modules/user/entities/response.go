package entities

// RegisterUserResponse is the outbound shape for POST /users. The brief's
// literal contract is a bare userId string; every other response in this
// service is a JSON object, so this wraps it the same way - see the
// project report for this deliberate deviation.
type RegisterUserResponse struct {
	UserID string `json:"userId"`
}
