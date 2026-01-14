package handler

import (
	"echo-backend/services/message-service/api/v1/dto"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// EditMessage handles editing an existing message
func (h *MessageHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Edit message request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
	)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Get message ID from path
	vars := mux.Vars(r)
	messageID := vars["id"]
	if messageID == "" {
		response.BadRequestError(r.Context(), r, w, "Message ID is required", nil)
		return
	}

	// Parse and validate request
	request := dto.NewEditMessageRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	// Call service layer
	err := h.service.EditMessage(r.Context(), uuid.MustParse(messageID), uuid.MustParse(userID), request.Content)
	if err != nil {
		h.log.Error("Failed to edit message",
			logger.String("user_id", userID),
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to edit message", err)
		return
	}

	h.log.Info("Message edited successfully",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
	)

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Message edited successfully", nil)
}
