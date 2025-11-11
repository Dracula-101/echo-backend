package handler

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/models"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	request := dto.NewGetMessagesRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}
	messages, err := h.service.GetMessages(r.Context(), uuid.MustParse(request.ConversationID), &models.PaginationParams{
		Limit: request.Limit,
	})
	if err != nil {
		h.log.Error("Failed to get messages",
			logger.String("conversation_id", request.ConversationID),
			logger.Error(err),
		)
		response.BadRequestError(r.Context(), r, w, "Failed to get messages", err)
		return
	}

	responseDTO := dto.GetMessagesResponse{
		MessagesResponse: *messages,
	}

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Messages retrieved successfully", responseDTO)

}
