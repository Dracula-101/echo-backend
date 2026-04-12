package domain

// MessageEventType represents the type of message event
type MessageEventType string

const (
	MessageEventTypeSent        MessageEventType = "message.sent"
	MessageEventTypeEdited      MessageEventType = "message.edited"
	MessageEventTypeDeleted     MessageEventType = "message.deleted"
	MessageEventTypeDelivered   MessageEventType = "message.delivered"
	MessageEventTypeRead        MessageEventType = "message.read"
	MessageEventTypeFailed      MessageEventType = "message.failed"
	MessageEventReactionAdded   MessageEventType = "message.reaction_added"
	MessageEventReactionRemoved MessageEventType = "message.reaction_removed"
	MessageEventPollCreated     MessageEventType = "message.poll_created"
	MessageEventPollVoted       MessageEventType = "message.poll_voted"
	MessageEventTypePinned      MessageEventType = "message.pinned"
	MessageEventTypeUnpinned    MessageEventType = "message.unpinned"
	MessageEventTypeForwarded   MessageEventType = "message.forwarded"
)

// ConversationEventType represents the type of conversation event
type ConversationEventType string

const (
	ConversationEventTypeCreated          ConversationEventType = "conversation.created"
	ConversationEventTypeUpdated          ConversationEventType = "conversation.updated"
	ConversationEventTypeDeleted          ConversationEventType = "conversation.deleted"
	ConversationEventTypeParticipantAdded ConversationEventType = "conversation.participant_added"
	ConversationEventTypeParticipantLeft  ConversationEventType = "conversation.participant_left"
	ConversationEventTypeTypingStarted    ConversationEventType = "conversation.typing_started"
	ConversationEventTypeTypingStopped    ConversationEventType = "conversation.typing_stopped"
)
