package message

import (
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// DeleteMessage flow at a glance:
//
//	[1] Request helper — capture request ID + auth context.
//	[2] Path params — pull message ID.
//	[3] MessageService.DeleteMessage — verify ownership/permissions, remove record.
//	[4] Response helper — respond 200 once deletion succeeds.
//
// Errors short-circuit with Unauthorized/BadRequest/500 as appropriate.
func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Delete message request received",
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
	messageID := handler.PathParam("id")

	// Call service layer
	err := h.messageService.DeleteMessage(r.Context(), uuid.MustParse(messageID), uuid.MustParse(userID))
	if err != nil {
		h.log.Error("Failed to delete message",
			logger.String("user_id", userID),
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to delete message", err)
		return
	}

	h.log.Info("Message deleted successfully",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
	)

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Message deleted successfully", nil)
}
