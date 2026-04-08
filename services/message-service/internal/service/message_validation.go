package service

import (
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	pkgErrors "shared/pkg/errors"
)

// handleValidationError maps conversation participant validation errors to MessageError codes.
func (s *messageService) handleValidationError(dbErr pkgErrors.AppError) *msgError.MessageError {
	switch dbErr.Code() {
	case msgError.CodeBlockedUserNotFound.String():
		return &msgError.MessageError{
			Message: "Participant not in conversation",
			Code:    msgError.CodeParticipantNotInConversation,
			Error:   dbErr,
		}
	case msgError.CodeParticipantLeftConversation.String():
		return &msgError.MessageError{
			Message: "Participant has left the conversation",
			Code:    msgError.CodeParticipantLeftConversation,
			Error:   dbErr,
		}
	case msgError.CodeParticipantRemovedFromConversation.String():
		return &msgError.MessageError{
			Message: "Participant has been removed from the conversation",
			Code:    msgError.CodeParticipantRemovedFromConversation,
			Error:   dbErr,
		}
	case msgError.CodeConversationInactive.String():
		return &msgError.MessageError{
			Message: "Conversation is inactive",
			Code:    msgError.CodeConversationInactive,
			Error:   dbErr,
		}
	case msgError.CodeParticipantNotAllowedToSendMessages.String():
		return &msgError.MessageError{
			Message: "Participant is not allowed to send messages",
			Code:    msgError.CodeParticipantNotAllowedToSendMessages,
			Error:   dbErr,
		}
	default:
		return &msgError.MessageError{
			Message: "Failed to validate conversation participant",
			Code:    msgError.CodeConversationValidationFailed,
			Error:   dbErr,
		}
	}
}

// validateMessage checks that the message type has the required metadata fields.
func (s *messageService) validateMessage(rawMsgType string, metadata *domain.MessageMetadata) *msgError.MessageError {
	msgType := domain.MessageType(rawMsgType)
	switch msgType {
	case domain.MessageTypeImage:
		if metadata == nil || metadata.MediaURL == "" {
			return &msgError.MessageError{
				Message: "Metadata with valid URL is required for image messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for image message"),
			}
		}
	case domain.MessageTypeVideo:
		if metadata == nil || metadata.MediaURL == "" {
			return &msgError.MessageError{
				Message: "Metadata with valid URL is required for video messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for video message"),
			}
		}
	case domain.MessageTypeFile:
		if metadata == nil || metadata.MediaURL == "" || metadata.FileName == "" {
			return &msgError.MessageError{
				Message: "Metadata with valid URL and filename is required for file messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for file message"),
			}
		}
	case domain.MessageTypeText:
		// No specific metadata validation for text messages
	case domain.MessageTypeAudio:
		if metadata == nil || metadata.MediaURL == "" {
			return &msgError.MessageError{
				Message: "Metadata with valid URL is required for audio messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for audio message"),
			}
		}
	case domain.MessageTypeLocation:
		if metadata == nil || metadata.Latitude == 0 || metadata.Longitude == 0 {
			return &msgError.MessageError{
				Message: "Metadata with valid latitude and longitude is required for location messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for location message"),
			}
		}
	default:
		return &msgError.MessageError{
			Message: "Unsupported message type",
			Code:    msgError.CodeMissingMessageMetadata,
			Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "unsupported message type"),
		}
	}
	return nil
}
