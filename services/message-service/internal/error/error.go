package error

import (
	"shared/pkg/errors"
	pkgErrors "shared/pkg/errors"
)

const ServiceName = "message_service"

type MessageErrorCode string

func (c MessageErrorCode) String() string {
	return string(c)
}

type MessageError struct {
	Code    MessageErrorCode
	Message string
	Error   pkgErrors.AppError
}

var (
	ConversationNotFoundError                = errors.New(CodeConversationNotFound.String(), "conversation not found")
	ParticipantNotInConversationError        = errors.New(CodeParticipantNotInConversation.String(), "participant not in conversation")
	ParticipantLeftConversationError         = errors.New(CodeParticipantLeftConversation.String(), "participant has left the conversation")
	ConversationInactiveError                = errors.New(CodeConversationInactive.String(), "conversation is inactive")
	ParticipantRemovedFromConversationError  = errors.New(CodeParticipantRemovedFromConversation.String(), "participant has been removed from the conversation")
	ParticipantNotAllowedToSendMessagesError = errors.New(CodeParticipantNotAllowedToSendMessages.String(), "participant is not allowed to send messages")
)
