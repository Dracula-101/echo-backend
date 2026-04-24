package handler

import (
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
)

func (h *UserHandler) GetSettings(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Received request to get settings",
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	// Cache hit: return without touching DB.
	cached, cacheErr := h.cache.GetFullSettings(ctx, userID)
	if cacheErr != nil {
		h.log.Error("Failed to get full settings from cache",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(cacheErr),
		)
	}
	if cached != nil {
		response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Settings retrieved successfully", cached)
		return
	}

	// Cache miss: query DB.
	s, err := h.settingsRepo.GetSettings(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get settings from DB",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to retrieve settings", nil)
		return
	}

	// First-time user: create a default row.
	if s == nil {
		s, err = h.settingsRepo.CreateSettings(ctx, userID)
		if err != nil {
			h.log.Error("Failed to create default settings",
				logger.String("request_id", requestID),
				logger.String("user_id", userID),
				logger.Error(err),
			)
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to retrieve settings", nil)
			return
		}
	}

	// Populate cache for subsequent reads (300 s TTL matches TTLSettings).
	if cacheErr := h.cache.SetFullSettings(ctx, userID, s); cacheErr != nil {
		h.log.Error("Failed to cache full settings",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(cacheErr),
		)
	}

	h.log.Info("Settings retrieved successfully",
		logger.String("request_id", requestID),
		logger.String("user_id", userID),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Settings retrieved successfully", s)
}
