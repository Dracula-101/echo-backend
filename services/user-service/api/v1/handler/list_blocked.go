package handler

import (
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
)

func (h *UserHandler) ListBlocked(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Received request to list blocked users",
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	requesterID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	blocked, err := h.blockedRepo.GetBlockedUsers(ctx, requesterID)
	if err != nil {
		h.log.Error("Failed to list blocked users",
			logger.String("request_id", requestID),
			logger.String("user_id", requesterID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to list blocked users", nil)
		return
	}

	h.log.Info("Blocked users retrieved successfully",
		logger.String("request_id", requestID),
		logger.String("user_id", requesterID),
		logger.Int("count", len(blocked)),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Blocked users retrieved successfully", dto.NewListBlockedResponse(blocked))
}
