package message

import (
	"net/http"

	"echo-backend/services/message-service/api/v1/dto"
	svcmodel "echo-backend/services/message-service/internal/service/model"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// GetMessages flow at a glance:
//
//	[1] Request helper — parse pagination + conversation ID, surface request ID.
//	[2] Auth context — ensure user ID available (gateway-authenticated).
//	[3] MessageService.GetMessages — fetch paginated history from storage.
//	[4] Response helper — wrap service DTO, return 200 JSON.
//
// All branches reuse the same trace IDs for observability.
func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Get messages request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
	)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	request := dto.NewGetMessagesRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	messages, err := h.messageService.GetMessages(r.Context(), uuid.MustParse(request.ConversationID), &svcmodel.PaginationParams{
		Limit: request.Limit,
	})
	if err != nil {
		h.log.Error("Failed to get messages",
			logger.String("conversation_id", request.ConversationID),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.BadRequestError(r.Context(), r, w, "Failed to get messages", err)
		return
	}

	responseDTO := dto.GetMessagesResponse{MessagesResponse: *messages}

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Messages retrieved successfully", responseDTO)
}
