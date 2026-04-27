package error

type MessageErrorCode string

const (
	CodeCacheError    = "MSG_CACHE_ERROR"
	CodeDBError       = "MSG_DB_ERROR"
	CodeInternalError = "MSG_INTERNAL_ERROR"
	CodeInvalidInput  = "MSG_INVALID_INPUT"

	CodeRateLimited = "MSG_RATE_LIMITED"

	CodeConversationAlreadyExists      = "MSG_CONVERSATION_ALREADY_EXISTS"
	CodeConversationCreationInProgress = "MSG_CONVERSATION_CREATION_IN_PROGRESS"
	CodeParticipantBlockedByOtherUser  = "MSG_PARTICIPANT_BLOCKED_BY_OTHER_USER"
	CodeNotParticipant                 = "MSG_NOT_PARTICIPANT"

	CodeMessageAlreadySent = "MSG_MESSAGE_ALREADY_SENT"
	CodeMessageTooLarge    = "MSG_MESSAGE_TOO_LARGE"
	CodeMessageDeleted     = "MSG_MESSAGE_DELETED"

	CodeReactionAlreadyExists = "MSG_REACTION_ALREADY_EXISTS"
	CodeReactionNotFound      = "MSG_REACTION_NOT_FOUND"

	CodeParticipantAlreadyExists = "MSG_PARTICIPANT_ALREADY_EXISTS"
	CodeParticipantNotFound      = "MSG_PARTICIPANT_NOT_FOUND"

	CodePinLimitExceeded = "MSG_PIN_LIMIT_EXCEEDED"

	CodeValidationError = "MSG_VALIDATION_ERROR"
	CodeUnauthorized    = "MSG_UNAUTHORIZED"
	CodeForbidden       = "MSG_FORBIDDEN"

	// User Errors
	CodeUserBlocked         MessageErrorCode = "USER_BLOCKED"
	CodeBlockedUserNotFound MessageErrorCode = "BLOCK_USER_NOT_FOUND"
	CodeRateLimitExceeded   MessageErrorCode = "RATE_LIMIT_EXCEEDED"

	// Idempotency Errors
	CodeIdempotencyKeyRequired MessageErrorCode = "IDEMPOTENCY_KEY_REQUIRED"
	CodeIdempotencyKeyInvalid  MessageErrorCode = "IDEMPOTENCY_KEY_INVALID"
	CodeIdempotencyConflict    MessageErrorCode = "IDEMPOTENCY_CONFLICT"

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
	CodeEmptyMessageContent        MessageErrorCode = "EMPTY_MESSAGE_CONTENT"
	CodeMissingMessageMetadata     MessageErrorCode = "MISSING_MESSAGE_METADATA"
	CodeMessageEventPublishFailed  MessageErrorCode = "MESSAGE_EVENT_PUBLISH_FAILED"
	CodeMessageRetrievalFailed     MessageErrorCode = "MESSAGE_RETRIEVAL_FAILED"
	CodeMessageNotFound            MessageErrorCode = "MESSAGE_NOT_FOUND"
	CodeMessageNotEditable         MessageErrorCode = "MESSAGE_NOT_EDITABLE"
	CodeMessageEditExpired         MessageErrorCode = "MESSAGE_EDIT_EXPIRED"
	CodeMessageEditFailed          MessageErrorCode = "MESSAGE_EDIT_FAILED"
	CodeMessageDeleteFailed        MessageErrorCode = "MESSAGE_DELETE_FAILED"
	CodeMessageNotDeletable        MessageErrorCode = "MESSAGE_NOT_DELETABLE"
	CodeForwardFailed              MessageErrorCode = "FORWARD_FAILED"
	CodeTargetConversationNotFound MessageErrorCode = "TARGET_CONVERSATION_NOT_FOUND"
	CodePinFailed                  MessageErrorCode = "PIN_FAILED"
	CodeAlreadyPinned              MessageErrorCode = "ALREADY_PINNED"
	CodeNotPinned                  MessageErrorCode = "NOT_PINNED"
	CodeSearchFailed               MessageErrorCode = "SEARCH_FAILED"
	CodeDeliveryStatusFailed       MessageErrorCode = "DELIVERY_STATUS_FAILED"
	CodeMarkAsReadFailed           MessageErrorCode = "MARK_AS_READ_FAILED"

	// Conversation Mutation Errors
	CodeConversationUpdateFailed MessageErrorCode = "CONVERSATION_UPDATE_FAILED"
	CodeConversationDeleteFailed MessageErrorCode = "CONVERSATION_DELETE_FAILED"
	CodePermissionDenied         MessageErrorCode = "PERMISSION_DENIED"
	CodeDirectConversationLimit  MessageErrorCode = "DIRECT_CONVERSATION_LIMIT"
	CodeMaxMembersExceeded       MessageErrorCode = "MAX_MEMBERS_EXCEEDED"
	CodeCannotRemoveOwner        MessageErrorCode = "CANNOT_REMOVE_OWNER"
)

func (c MessageErrorCode) String() string {
	return string(c)
}
