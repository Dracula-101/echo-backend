package handler

import (
	"net/http"

	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *UserHandler) UnblockUser(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Received request to unblock user",
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	targetIDStr := handler.PathParam("user_id")
	if _, err := uuid.Parse(targetIDStr); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invalid user ID format", nil)
		return
	}

	requesterID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	block, err := h.blockedRepo.GetBlockedUser(ctx, requesterID, targetIDStr)
	if err != nil {
		h.log.Error("Failed to look up block record",
			logger.String("request_id", requestID),
			logger.String("target_id", targetIDStr),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process unblock request", nil)
		return
	}
	if block == nil {
		response.NotFoundError(ctx, handler.Request(), handler.Writer(), "block record")
		return
	}

	if err := h.blockedRepo.UnblockUser(ctx, requesterID, targetIDStr); err != nil {
		h.log.Error("Failed to unblock user",
			logger.String("request_id", requestID),
			logger.String("target_id", targetIDStr),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to unblock user", nil)
		return
	}

	// TODO: publish user.unblocked to Kafka

	// Cache: remove target from requester's blocked set.
	if cacheErr := h.cache.RemoveBlockedID(ctx, requesterID, targetIDStr); cacheErr != nil {
		h.log.Error("Failed to remove blocked ID from cache",
			logger.String("request_id", requestID),
			logger.String("target_id", targetIDStr),
			logger.Error(cacheErr),
		)
	}

	// Unblock changes the privacy relationship for both parties.
	if cacheErr := h.cache.DeletePrivacy(ctx, requesterID, targetIDStr); cacheErr != nil {
		h.log.Error("Failed to invalidate privacy cache (requester→target)",
			logger.String("request_id", requestID),
			logger.Error(cacheErr),
		)
	}
	if cacheErr := h.cache.DeletePrivacy(ctx, targetIDStr, requesterID); cacheErr != nil {
		h.log.Error("Failed to invalidate privacy cache (target→requester)",
			logger.String("request_id", requestID),
			logger.Error(cacheErr),
		)
	}

	h.log.Info("User unblocked successfully",
		logger.String("request_id", requestID),
		logger.String("target_id", targetIDStr),
	)

	handler.Writer().WriteHeader(http.StatusNoContent)
}
