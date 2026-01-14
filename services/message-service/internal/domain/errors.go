package domain

import "errors"

// Domain-level errors for message service
var (
	// Message errors
	ErrMessageNotFound       = errors.New("message not found")
	ErrMessageDeleted        = errors.New("message has been deleted")
	ErrMessageNotEditable    = errors.New("message cannot be edited")
	ErrMessageEditExpired    = errors.New("edit time window has expired")
	ErrCannotDeleteMessage   = errors.New("you cannot delete this message")
	ErrInvalidMessageType    = errors.New("invalid message type")
	ErrEmptyMessageContent   = errors.New("message content cannot be empty")
	ErrMessageTooLong        = errors.New("message content exceeds maximum length")
	ErrInvalidMediaURL       = errors.New("invalid media URL")

	// Conversation errors
	ErrConversationNotFound  = errors.New("conversation not found")
	ErrConversationDeleted   = errors.New("conversation has been deleted")
	ErrConversationArchived  = errors.New("conversation is archived")
	ErrInvalidConversationType = errors.New("invalid conversation type")
	ErrConversationTitleRequired = errors.New("conversation title is required for groups")
	ErrDuplicateConversation = errors.New("conversation already exists")

	// Participant errors
	ErrNotParticipant        = errors.New("user is not a participant in this conversation")
	ErrAlreadyParticipant    = errors.New("user is already a participant")
	ErrNoSendPermission      = errors.New("you don't have permission to send messages")
	ErrNoEditPermission      = errors.New("you don't have permission to edit conversation")
	ErrNoDeletePermission    = errors.New("you don't have permission to delete messages")
	ErrCannotLeaveConversation = errors.New("conversation owner cannot leave")
	ErrInsufficientParticipants = errors.New("conversation must have at least 2 participants")
	ErrTooManyParticipants   = errors.New("conversation has reached maximum participants")

	// Delivery errors
	ErrDeliveryStatusNotFound = errors.New("delivery status not found")
	ErrAlreadyMarkedAsRead   = errors.New("message already marked as read")
	ErrCannotMarkAsRead      = errors.New("cannot mark message as read")

	// Validation errors
	ErrInvalidUserID         = errors.New("invalid user ID")
	ErrInvalidConversationID = errors.New("invalid conversation ID")
	ErrInvalidMessageID      = errors.New("invalid message ID")
	ErrInvalidPagination     = errors.New("invalid pagination parameters")

	// Business rule errors
	ErrTypingIndicatorExpired = errors.New("typing indicator has expired")
	ErrSelfMessage           = errors.New("cannot send message to yourself in direct conversation")
)
