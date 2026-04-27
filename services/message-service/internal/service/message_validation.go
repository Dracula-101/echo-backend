package service

import (
	"strings"
	"time"

	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	pkgErrors "shared/pkg/errors"
)

// handleValidationError maps conversation participant validation errors to MsgError codes.
func (s *messageService) handleValidationError(dbErr pkgErrors.AppError) *msgError.MsgError {
	switch dbErr.Code() {
	case msgError.CodeBlockedUserNotFound.String():
		return &msgError.MsgError{
			Message: "Participant not in conversation",
			Code:    msgError.CodeParticipantNotInConversation,
			Error:   dbErr,
		}
	case msgError.CodeParticipantLeftConversation.String():
		return &msgError.MsgError{
			Message: "Participant has left the conversation",
			Code:    msgError.CodeParticipantLeftConversation,
			Error:   dbErr,
		}
	case msgError.CodeParticipantRemovedFromConversation.String():
		return &msgError.MsgError{
			Message: "Participant has been removed from the conversation",
			Code:    msgError.CodeParticipantRemovedFromConversation,
			Error:   dbErr,
		}
	case msgError.CodeConversationInactive.String():
		return &msgError.MsgError{
			Message: "Conversation is inactive",
			Code:    msgError.CodeConversationInactive,
			Error:   dbErr,
		}
	case msgError.CodeParticipantNotAllowedToSendMessages.String():
		return &msgError.MsgError{
			Message: "Participant is not allowed to send messages",
			Code:    msgError.CodeParticipantNotAllowedToSendMessages,
			Error:   dbErr,
		}
	default:
		return &msgError.MsgError{
			Message: "Failed to validate conversation participant",
			Code:    msgError.CodeConversationValidationFailed,
			Error:   dbErr,
		}
	}
}

func (s *messageService) validateSendInput(req *domain.SendMessageInput) *msgError.MsgError {
	switch req.MessageType {
	case domain.MessageTypeText:
		if strings.TrimSpace(req.Content) == "" {
			return missingField("Text messages require non-empty content", "missing content for text message")
		}
	case domain.MessageTypeImage, domain.MessageTypeVideo, domain.MessageTypeAudio,
		domain.MessageTypeDocument, domain.MessageTypeFile:
		if len(req.MediaIDs) == 0 {
			return missingField("media_ids is required for media messages", "missing media_ids")
		}
	case domain.MessageTypeVoiceNote:
		if len(req.MediaIDs) != 1 {
			return missingField("voice_note must reference exactly one media file", "invalid media_ids count for voice_note")
		}
	case domain.MessageTypeSticker:
		if req.StickerID == nil {
			return missingField("sticker_id is required for sticker messages", "missing sticker_id")
		}
	case domain.MessageTypeGif:
		if req.GifID == nil {
			return missingField("gif_id is required for gif messages", "missing gif_id")
		}
	case domain.MessageTypeLocation:
		if req.Location == nil {
			return missingField("location is required for location messages", "missing location")
		}
	case domain.MessageTypeContact:
		if req.Contact == nil || req.Contact.Name == "" {
			return missingField("contact with at least a name is required for contact messages", "missing contact")
		}
		if req.Contact.Phone == "" && req.Contact.Email == "" {
			return missingField("contact requires at least one of phone or email", "missing contact phone/email")
		}
	case domain.MessageTypePoll:
		return missingField("polls must be created via POST /polls", "poll cannot be sent through messages endpoint")
	default:
		return missingField("Unsupported message type", "unsupported message type")
	}

	if req.ScheduledAt != nil {
		now := time.Now()
		if !req.ScheduledAt.After(now) {
			return missingField("scheduled_at must be in the future", "scheduled_at not in future")
		}
		if req.ScheduledAt.After(now.Add(7 * 24 * time.Hour)) {
			return missingField("scheduled_at cannot be more than 7 days ahead", "scheduled_at exceeds 7 day window")
		}
	}

	return nil
}

func missingField(userMsg, internalMsg string) *msgError.MsgError {
	return &msgError.MsgError{
		Message: userMsg,
		Code:    msgError.CodeMissingMessageMetadata,
		Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, internalMsg),
	}
}
