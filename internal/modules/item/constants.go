package item

// Log messages are static strings; dynamic detail always travels as
// key-value pairs on the same call - see go-coding-standards.
const (
	logMsgItemCreated      = "item created"
	logMsgItemUpdated      = "item updated"
	logMsgItemDeleted      = "item deleted"
	logMsgCreateItemFailed = "failed to create item"
	logMsgGetItemFailed    = "failed to get item"
	logMsgListItemsFailed  = "failed to list items"
	logMsgUpdateItemFailed = "failed to update item"
	logMsgDeleteItemFailed = "failed to delete item"

	logFieldError  = "error"
	logFieldItemID = "itemID"
)
