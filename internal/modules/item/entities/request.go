package entities

// Struct tags must be literals, so the max values below cannot reference
// MaxNameLength/MaxDescriptionLength directly. They are a first pass only:
// core.go re-checks every write against those constants, so a drifted tag
// cannot let a bad value reach the database.

// CreateItemRequest is the inbound shape for POST /api/items. Gin's binding
// tags handle presence and length checks.
type CreateItemRequest struct {
	Name        string `json:"name" binding:"required,max=200"`
	Description string `json:"description" binding:"max=2000"`
}

// UpdateItemRequest is the inbound shape for PUT /api/items/:id.
type UpdateItemRequest struct {
	Name        string `json:"name" binding:"required,max=200"`
	Description string `json:"description" binding:"max=2000"`
}
