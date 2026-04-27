package message

import (
	"encoding/json"

	"echo-backend/services/message-service/api/v1/dto"
	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/service/idempotency"

	"shared/pkg/logger"
	"shared/server/headers"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) ForwardMessage(handler *req.RequestHandler) {
	ctx := handler.Context()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	messageID := handler.PathParam("id")
	if _, err := uuid.Parse(messageID); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Message ID must be a valid UUID", err)
		return
	}

	idempotencyKey := handler.Request().Header.Get(headers.XIdempotencyKey)
	if idempotencyKey == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), string(msgError.CodeIdempotencyKeyRequired), nil)
		return
	}
	if _, err := uuid.Parse(idempotencyKey); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), string(msgError.CodeIdempotencyKeyInvalid), err)
		return
	}

	idemRes, idemErr := h.idempotency.CheckAndAcquire(ctx, idempotency.ScopeForwardMessage, idempotencyKey)
	if idemErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), idemErr.Message, idemErr.Error)
		return
	}
	if idemRes.Hit && idemRes.InFlight {
		response.ConflictError(ctx, handler.Request(), handler.Writer(),
			"A forward with this idempotency key is already in progress", nil)
		return
	}
	if idemRes.Hit && len(idemRes.Payload) > 0 {
		var cached dto.ForwardMessageResponse
		if err := json.Unmarshal(idemRes.Payload, &cached); err == nil {
			response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
				response.StatusOK, "Message already forwarded", &cached)
			return
		}
	}

	body := dto.NewForwardMessageRequest()
	if !handler.ParseValidateAndSend(body) {
		_ = h.idempotency.Release(ctx, idempotency.ScopeForwardMessage, idempotencyKey)
		return
	}

	results := make([]dto.ForwardedTarget, 0, len(body.ConversationIDs))
	deviceInfo := handler.GetDeviceInfo()
	clientIP := handler.GetClientIP()

	for _, target := range body.ConversationIDs {
		msg, fwdErr := h.messageService.ForwardMessage(ctx, messageID, userID, target, deviceInfo, clientIP)
		if fwdErr != nil {
			_ = h.idempotency.Release(ctx, idempotency.ScopeForwardMessage, idempotencyKey)
			h.log.Error("Failed to forward message",
				logger.String("user_id", userID),
				logger.String("message_id", messageID),
				logger.String("target_conversation_id", target),
				logger.String("code", fwdErr.Code.String()),
			)
			switch fwdErr.Code {
			case msgError.CodeMessageNotFound:
				response.NotFoundError(ctx, handler.Request(), handler.Writer(), fwdErr.Message)
			case msgError.CodeTargetConversationNotFound, msgError.CodeMessageNotDeletable:
				response.BadRequestError(ctx, handler.Request(), handler.Writer(), fwdErr.Message, fwdErr.Error)
			default:
				response.InternalServerError(ctx, handler.Request(), handler.Writer(), fwdErr.Message, fwdErr.Error)
			}
			return
		}
		results = append(results, dto.ForwardedTarget{
			ConversationID: msg.ConversationID.String(),
			MessageID:      msg.ID.String(),
		})
	}

	resp := &dto.ForwardMessageResponse{ForwardedTo: results}
	if encoded, err := json.Marshal(resp); err == nil {
		_ = h.idempotency.Store(ctx, idempotency.ScopeForwardMessage, idempotencyKey, encoded)
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusCreated, "Message forwarded", resp)
}
