package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// GetConversations flow at a glance:
//
//	[1] Request helper — parse pagination filters, expose request ID.
//	[2] Auth context — ensure owner user ID.
//	[3] ConversationService.GetConversations — fetch list + total count.
//	[4] Response helper — emit 200 with paging metadata (hasMore, total, etc.).
//
// Shared request IDs make each branch traceable through logs.
func (h *ConversationHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Get conversations request received",
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
	request := dto.NewGetConversationsRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	// Call service layer
	conversations, total, err := h.service.GetConversations(
		uuid.MustParse(userID),
		request.Limit,
		request.Offset,
	)

	if err != nil {
		h.log.Error("Failed to get conversations",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to get conversations", err)
		return
	}

	hasMore := request.Offset+len(conversations) < total

	// Send response
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Conversations retrieved successfully",
		dto.GetConversationsResponse{
			Conversations: conversations,
			Total:         total,
			Limit:         request.Limit,
			Offset:        request.Offset,
			HasMore:       hasMore,
		},
	)
}
