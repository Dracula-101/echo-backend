package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// CreateConversation flow at a glance:
//
//	[1] Request helper — parse payload, capture request/correlation IDs.
//	[2] Auth context — ensure creator user ID.
//	[3] Payload shaping — coerce participant IDs into uuid.UUID slice.
//	[4] ConversationService.CreateConversation — persist convo + participants.
//	[5] Response helper — return 201 DTO with metadata.
//
// Any validation/service failures short-circuit with structured errors.
func (h *ConversationHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Create conversation request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		h.log.Warn("User not authenticated",
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Parse and validate request
	request := dto.NewCreateConversationRequest()
	if !handler.ParseValidateAndSend(request) {
		h.log.Warn("Create conversation validation failed",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
		)
		return
	}

	h.log.Debug("Creating conversation",
		logger.String("user_id", userID),
		logger.String("conversation_type", request.ConversationType),
		logger.Int("participant_count", len(request.ParticipantIDs)),
	)

	// Parse participant IDs
	participantIDs := make([]uuid.UUID, len(request.ParticipantIDs))
	for i, id := range request.ParticipantIDs {
		participantIDs[i] = uuid.MustParse(id)
	}

	// Call service layer
	conversationID, allParticipants, createdAt, err := h.service.CreateConversation(
		uuid.MustParse(userID),
		request.ConversationType,
		participantIDs,
		request.Title,
		request.Description,
		request.IsEncrypted,
		request.IsPublic,
	)

	if err != nil {
		h.log.Error("Failed to create conversation",
			logger.String("user_id", userID),
			logger.String("conversation_type", request.ConversationType),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to create conversation", err)
		return
	}

	h.log.Info("Conversation created successfully",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID.String()),
		logger.String("conversation_type", request.ConversationType),
	)

	// Send response
	response.JSONWithMessage(r.Context(), r, w, http.StatusCreated, "Conversation created successfully",
		dto.NewCreateConversationResponse(
			conversationID,
			request.ConversationType,
			request.Title,
			request.Description,
			uuid.MustParse(userID),
			request.IsEncrypted,
			request.IsPublic,
			allParticipants,
			createdAt,
		),
	)
}
