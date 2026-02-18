package protocol

type MessageFilters struct {
	UserID          *string   `json:"user_id,omitempty"`
	ConversationID  *string   `json:"conversation_id,omitempty"`
	ConversationIDs *[]string `json:"conversation_ids,omitempty"`
}
