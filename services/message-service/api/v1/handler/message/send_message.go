package message

import (
	"encoding/json"

	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/service/idempotency"

	"shared/pkg/logger"
	"shared/server/headers"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) SendMessage(handler *request.RequestHandler) {
	requestID := handler.GetRequestID()

	userIDStr, ok := request.GetUserIDFromContext(handler.Context())
	if !ok {
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Invalid user id", err)
		return
	}

	convIDStr := handler.PathParam("conversation_id")
	conversationID, err := uuid.Parse(convIDStr)
	if err != nil {
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "conversation_id must be a valid UUID", err)
		return
	}

	idempotencyKey := handler.Request().Header.Get(headers.XIdempotencyKey)
	if idempotencyKey == "" {
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(),
			string(msgError.CodeIdempotencyKeyRequired), nil)
		return
	}
	if _, parseErr := uuid.Parse(idempotencyKey); parseErr != nil {
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(),
			string(msgError.CodeIdempotencyKeyInvalid), parseErr)
		return
	}

	idemRes, idemErr := h.idempotency.CheckAndAcquire(handler.Context(), idempotency.ScopeSendMessage, idempotencyKey)
	if idemErr != nil {
		response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), idemErr.Message, idemErr.Error)
		return
	}
	if idemRes.Hit && idemRes.InFlight {
		response.ConflictError(handler.Context(), handler.Request(), handler.Writer(),
			"A request with this idempotency key is already in progress", nil)
		return
	}
	if idemRes.Hit && len(idemRes.Payload) > 0 {
		var cached dto.SendMessageResponse
		if err := json.Unmarshal(idemRes.Payload, &cached); err != nil {
			h.log.Warn("idempotency cache payload unmarshal failed; replaying as raw",
				logger.String("request_id", requestID),
				logger.Error(err),
			)
			response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(),
				response.StatusOK, "Message already sent", json.RawMessage(idemRes.Payload),
			)
			return
		}
		response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(),
			response.StatusOK, "Message already sent", &cached,
		)
		return
	}

	req := dto.NewSendMessageRequest()
	if !handler.ParseValidateAndSend(req) {
		_ = h.idempotency.Release(handler.Context(), idempotency.ScopeSendMessage, idempotencyKey)
		return
	}

	input, mapErr := buildSendMessageInput(req, conversationID, userID, idempotencyKey)
	if mapErr != nil {
		_ = h.idempotency.Release(handler.Context(), idempotency.ScopeSendMessage, idempotencyKey)
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), mapErr.Error(), mapErr)
		return
	}

	msg, svcErr := h.messageService.SendMessage(
		handler.Context(),
		input,
		handler.GetDeviceInfo(),
		handler.GetClientIP(),
	)
	if svcErr != nil {
		_ = h.idempotency.Release(handler.Context(), idempotency.ScopeSendMessage, idempotencyKey)
		h.log.Error("Failed to send message",
			logger.String("request_id", requestID),
			logger.String("user_id", userIDStr),
			logger.String("conversation_id", convIDStr),
			logger.String("code", svcErr.Code.String()),
			logger.Error(svcErr.Error),
		)
		switch svcErr.Code {
		case msgError.CodeUserBlocked, msgError.CodeParticipantNotAllowedToSendMessages, msgError.CodeConversationInactive:
			response.ForbiddenError(handler.Context(), handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeRateLimitExceeded:
			response.TooManyRequestsError(handler.Context(), handler.Request(), handler.Writer(), svcErr.Message, 60)
		case msgError.CodeIdempotencyConflict:
			response.ConflictError(handler.Context(), handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeParticipantNotInConversation, msgError.CodeBlockedUserNotFound, msgError.CodeParticipantRemovedFromConversation,
			msgError.CodeParticipantLeftConversation, msgError.CodeEmptyMessageContent, msgError.CodeMissingMessageMetadata, msgError.CodeConversationValidationFailed:
			response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		default:
			response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		}
		return
	}

	resp := dto.NewSendMessageResponse(msg)
	if encoded, marshalErr := json.Marshal(resp); marshalErr == nil {
		if storeErr := h.idempotency.Store(handler.Context(), idempotency.ScopeSendMessage, idempotencyKey, encoded); storeErr != nil {
			h.log.Warn("idempotency store failed; result not cached",
				logger.String("request_id", requestID),
				logger.Error(storeErr.Error),
			)
		}
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(),
		response.StatusCreated, "Message sent", resp,
	)
}

func buildSendMessageInput(req *dto.SendMessageRequest, conversationID, senderUserID uuid.UUID, idempotencyKey string) (*domain.SendMessageInput, error) {
	input := &domain.SendMessageInput{
		ConversationID: conversationID,
		SenderUserID:   senderUserID,
		Content:        req.Content,
		MessageType:    req.MessageType,
		ScheduledAt:    req.ScheduledAt,
		ExpireAfter:    req.ExpireAfter,
		IdempotencyKey: idempotencyKey,
	}

	if req.ParentMessageID != nil {
		id, err := uuid.Parse(*req.ParentMessageID)
		if err != nil {
			return nil, err
		}
		input.ParentMessageID = &id
	}
	if req.ForwardedFrom != nil {
		id, err := uuid.Parse(*req.ForwardedFrom)
		if err != nil {
			return nil, err
		}
		input.ForwardedFrom = &id
	}
	if req.StickerID != nil {
		id, err := uuid.Parse(*req.StickerID)
		if err != nil {
			return nil, err
		}
		input.StickerID = &id
	}
	if req.GifID != nil {
		id, err := uuid.Parse(*req.GifID)
		if err != nil {
			return nil, err
		}
		input.GifID = &id
	}

	if len(req.MediaIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(req.MediaIDs))
		for _, s := range req.MediaIDs {
			id, err := uuid.Parse(s)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		input.MediaIDs = ids
	}

	if len(req.Mentions) > 0 {
		mentions := make([]domain.Mention, 0, len(req.Mentions))
		for _, m := range req.Mentions {
			uid, err := uuid.Parse(m.UserID)
			if err != nil {
				return nil, err
			}
			mentions = append(mentions, domain.Mention{
				UserID:   uid,
				Username: m.Username,
				Offset:   m.Offset,
				Length:   m.Length,
			})
		}
		input.Mentions = mentions
	}

	if req.Location != nil {
		input.Location = &domain.Location{
			Latitude:  req.Location.Latitude,
			Longitude: req.Location.Longitude,
			Name:      req.Location.Name,
			Address:   req.Location.Address,
		}
	}
	if req.Contact != nil {
		input.Contact = &domain.Contact{
			Name:  req.Contact.Name,
			Phone: req.Contact.Phone,
			Email: req.Contact.Email,
		}
	}

	return input, nil
}
