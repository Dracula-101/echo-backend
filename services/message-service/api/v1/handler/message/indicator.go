package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// SetTypingIndicator flow at a glance:
//
//	[1] Request helper — parse DTO with conversation ID + typing state.
//	[2] Auth context — ensure user ID exists.
//	[3] MessageService.SetTypingIndicator — persist/broadcast state change.
//	[4] Response helper — emit 200 confirmation.
//
// Trace IDs + user/conversation IDs are logged for debugging.
func (h *MessageHandler) SetTypingIndicator(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Parse and validate request
	request := dto.NewTypingIndicatorRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	// Call service layer
	err := h.messageService.SetTypingIndicator(r.Context(), uuid.MustParse(request.ConversationID), uuid.MustParse(userID), request.IsTyping)
	if err != nil {
		h.log.Error("Failed to set typing indicator",
			logger.String("user_id", userID),
			logger.String("conversation_id", request.ConversationID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to set typing indicator", err)
		return
	}

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Typing indicator set", nil)
}
