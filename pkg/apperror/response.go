package apperror

// Response is the JSON wire format sent to API clients. Fields is omitted
// entirely when empty rather than serialized as null or {}.
type Response struct {
	Code    Code              `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Response converts the error to its wire format. This is the only place
// that decides what an *Error exposes to a client - internal is never
// touched here.
func (e *Error) Response() Response {
	return Response{Code: e.Code, Message: e.Message, Fields: e.Fields}
}
