package handler

import (
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"time"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
	userErrors "user-service/internal/errors"
)

func (h *UserHandler) UpdateContact(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestId := handler.GetRequestID()
	correlationId := handler.GetCorrelationID()

	h.log.Info("Received request to update contact information",
		logger.String("requestId", requestId),
		logger.String("correlationId", correlationId),
	)

	updateDto := dto.NewUpdateContactRequest()
	ok := handler.ParseValidateAndSend(updateDto)
	if !ok {
		h.log.Error("Failed to parse and validate update contact request",
			logger.String("requestId", requestId),
			logger.String("correlationId", correlationId),
		)
		return
	}

	requesterUserId, ok := request.GetUserIDUUIDFromContext(ctx)
	if !ok {
		h.log.Error("Failed to get user ID from context",
			logger.String("requestId", requestId),
			logger.String("correlationId", correlationId),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	contactId := handler.PathParam("contact_id")
	var mutedUntil *time.Time
	if updateDto.MutedUntil != nil {
		parsedMutedUntil, err := time.Parse(time.RFC3339, *updateDto.MutedUntil)
		if err != nil {
			h.log.Error("Failed to parse muted_until field",
				logger.String("requestId", requestId),
				logger.String("correlationId", correlationId),
				logger.Error(err),
			)
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Muted until must be a valid ISO 8601 datetime", nil)
			return
		}
		mutedUntil = &parsedMutedUntil
	}

	err := h.contactService.UpdateContact(ctx, requesterUserId.String(), contactId, domain.UpdateContact{
		Nickname:      updateDto.Nickname,
		Notes:         updateDto.Notes,
		IsFavorite:    updateDto.IsFavorite,
		IsPinned:      updateDto.IsPinned,
		IsArchived:    updateDto.IsArchived,
		IsMuted:       updateDto.IsMuted,
		MutedUntil:    mutedUntil,
		ContactGroups: updateDto.ContactGroups,
	})
	if err != nil {
		h.log.Error("Failed to update contact information",
			logger.String("requestId", requestId),
			logger.String("correlationId", correlationId),
			logger.Error(err),
		)
		if pkgErrors.GetCode(err) == userErrors.ErrCodeInvalidRequest {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact not found or not in active state", err)
			return
		}
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to update contact information", err)
		return
	}

	h.log.Info("Contact information updated successfully",
		logger.String("requestId", requestId),
		logger.String("correlationId", correlationId),
		logger.String("user_id", requesterUserId.String()),
		logger.String("contact_id", contactId),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Contact information updated successfully", nil)
}
