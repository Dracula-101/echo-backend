package domain

import "github.com/google/uuid"

// SendMessageResult contains the result of sending a message
type SendMessageResult struct {
	Message           *Message
	DeliveryStatuses  []DeliveryStatus
	OnlineRecipients  []uuid.UUID
	OfflineRecipients []uuid.UUID
}

// GetMessagesResult contains the result of fetching messages
type GetMessagesResult struct {
	Messages   []Message
	TotalCount int
	HasMore    bool
	NextCursor *uuid.UUID
	PrevCursor *uuid.UUID
}

// CreateConversationResult contains the result of creating a conversation
type CreateConversationResult struct {
	Conversation *Conversation
	Participants []ConversationParticipant
}

// GetConversationsResult contains the result of fetching conversations
type GetConversationsResult struct {
	Conversations []Conversation
	TotalCount    int
	TotalUnread   int
}
