package error

const (
	// User Errors
	CodeUserBlocked         MessageErrorCode = "USER_BLOCKED"
	CodeBlockedUserNotFound MessageErrorCode = "BLOCK_USER_NOT_FOUND"
	CodeRateLimitExceeded   MessageErrorCode = "RATE_LIMIT_EXCEEDED"

	// Conversation Errors
	CodeConversationNotFound         MessageErrorCode = "CONVERSATION_NOT_FOUND"
	CodeConversationValidationFailed MessageErrorCode = "CONVERSATION_VALIDATION_FAILED"
	CodeConversationInactive         MessageErrorCode = "CONVERSATION_INACTIVE"
	CodeConversationCreateFailed     MessageErrorCode = "CONVERSATION_CREATE_FAILED"
	CodeConversationCreationFailed   MessageErrorCode = "CONVERSATION_CREATION_FAILED"
	CodeConversationFetchFailed      MessageErrorCode = "CONVERSATION_FETCH_FAILED"

	// Participant Errors
	CodeParticipantNotInConversation        MessageErrorCode = "PARTICIPANT_NOT_IN_CONVERSATION"
	CodeParticipantLeftConversation         MessageErrorCode = "PARTICIPANT_LEFT_CONVERSATION"
	CodeParticipantRemovedFromConversation  MessageErrorCode = "PARTICIPANT_REMOVED_FROM_CONVERSATION"
	CodeParticipantNotAllowedToSendMessages MessageErrorCode = "PARTICIPANT_NOT_ALLOWED_TO_SEND_MESSAGES"

	// Message Errors
	CodeEmptyMessageContent       MessageErrorCode = "EMPTY_MESSAGE_CONTENT"
	CodeMissingMessageMetadata    MessageErrorCode = "MISSING_MESSAGE_METADATA"
	CodeMessageEventPublishFailed MessageErrorCode = "MESSAGE_EVENT_PUBLISH_FAILED"
	CodeMessageRetrievalFailed    MessageErrorCode = "MESSAGE_RETRIEVAL_FAILED"
)
