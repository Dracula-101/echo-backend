package handler

import (
	"net/http"

	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
	userErrors "user-service/internal/errors"
)

func (h *UserHandler) BlockUser(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Received request to block user",
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	blockDto := dto.NewBlockUserRequest()
	if ok := handler.ParseValidateAndSend(blockDto); !ok {
		return
	}

	requesterID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	targetID := blockDto.UserID

	if targetID == requesterID {
		h.log.Info("User attempted to block themselves",
			logger.String("request_id", requestID),
			logger.String("user_id", requesterID),
		)
		response.Error().
			WithContext(ctx).
			WithRequest(handler.Request()).
			WithMessage("Cannot block yourself").
			UnprocessableEntity(handler.Writer())
		return
	}

	// Check cache first; nil means a cache miss — proceed and let the DB check handle it.
	cachedBlockedIDs, cacheErr := h.cache.GetBlockedIDs(ctx, requesterID)
	if cacheErr != nil {
		h.log.Error("Failed to get blocked IDs from cache",
			logger.String("request_id", requestID),
			logger.Error(cacheErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process block request", nil)
		return
	}
	if cachedBlockedIDs != nil {
		for _, id := range *cachedBlockedIDs {
			if id == targetID {
				h.log.Info("User is already blocked (cache hit)",
					logger.String("request_id", requestID),
					logger.String("target_id", targetID),
				)
				response.ConflictError(ctx, handler.Request(), handler.Writer(), "User is already blocked", nil)
				return
			}
		}
	}

	// Acquire distributed lock to prevent concurrent block requests for the same pair.
	acquired, lockErr := h.cache.AcquireBlockLock(ctx, requesterID, targetID)
	if lockErr != nil {
		h.log.Error("Failed to acquire block lock",
			logger.String("request_id", requestID),
			logger.Error(lockErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process block request", nil)
		return
	}
	if !acquired {
		h.log.Info("Block lock already held for user pair",
			logger.String("request_id", requestID),
		)
		response.LockedError(ctx, handler.Request(), handler.Writer(), "block request")
		return
	}
	defer func() {
		if err := h.cache.ReleaseBlockLock(ctx, requesterID, targetID); err != nil {
			h.log.Error("Failed to release block lock",
				logger.String("request_id", requestID),
				logger.Error(err),
			)
		}
	}()

	blockType := blockDto.BlockType
	if blockType == "" {
		blockType = "full"
	}

	blockID, blockErr := h.blockedRepo.BlockUser(ctx, requesterID, targetID, blockDto.BlockReason, blockType)
	if blockErr != nil {
		if blockErr.Code() == userErrors.ErrCodeUserAlreadyBlocked {
			response.ConflictError(ctx, handler.Request(), handler.Writer(), "User is already blocked", nil)
			return
		}
		h.log.Error("Failed to block user",
			logger.String("request_id", requestID),
			logger.String("target_id", targetID),
			logger.Error(blockErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to block user", nil)
		return
	}

	// TODO: INSERT INTO users.outbox — user.blocked event with {blocker_id, blocked_id, block_type}

	// Cache: add to blocker's blocked set.
	if err := h.cache.AddBlockedID(ctx, requesterID, targetID); err != nil {
		h.log.Error("Failed to update blocked IDs cache",
			logger.String("request_id", requestID),
			logger.Error(err),
		)
	}

	// Cache: force-rebuild contacts list so blocked user no longer appears.
	if err := h.cache.DeleteContactIDs(ctx, requesterID); err != nil {
		h.log.Error("Failed to invalidate contact IDs cache",
			logger.String("request_id", requestID),
			logger.Error(err),
		)
	}

	// Cache: invalidate privacy cache for both directions.
	if err := h.cache.DeletePrivacy(ctx, requesterID, targetID); err != nil {
		h.log.Error("Failed to invalidate privacy cache (requester→target)",
			logger.String("request_id", requestID),
			logger.Error(err),
		)
	}
	if err := h.cache.DeletePrivacy(ctx, targetID, requesterID); err != nil {
		h.log.Error("Failed to invalidate privacy cache (target→requester)",
			logger.String("request_id", requestID),
			logger.Error(err),
		)
	}

	// Cache: remove blocked user from requester's contact presence feed.
	if err := h.cache.DeleteContactPresence(ctx, requesterID); err != nil {
		h.log.Error("Failed to invalidate contact presence cache",
			logger.String("request_id", requestID),
			logger.Error(err),
		)
	}

	// TODO: Kafka consumers — message-service closes/hides shared DM conversations,
	// ws-service removes blocked user from real-time channel state,
	// notification-service suppresses pending notifications from blocked user.

	h.log.Info("User blocked successfully",
		logger.String("request_id", requestID),
		logger.String("target_id", targetID),
		logger.String("block_id", blockID),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), http.StatusOK, "User blocked successfully", map[string]interface{}{
		"blocked":  true,
		"block_id": blockID,
	})
}
