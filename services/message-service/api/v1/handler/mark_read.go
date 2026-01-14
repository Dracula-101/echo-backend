package handler

import (
	"echo-backend/services/message-service/api/v1/dto"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// MarkAsRead handles marking a message as read
func (h *MessageHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Mark as read request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
	)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Parse and validate request
	request := dto.NewMarkAsReadRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	// Call service layer
	err := h.service.MarkAsRead(r.Context(), uuid.MustParse(request.MessageID), uuid.MustParse(userID))
	if err != nil {
		h.log.Error("Failed to mark message as read",
			logger.String("user_id", userID),
			logger.String("message_id", request.MessageID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to mark as read", err)
		return
	}

	h.log.Info("Message marked as read",
		logger.String("user_id", userID),
		logger.String("message_id", request.MessageID),
	)

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Message marked as read", nil)
}