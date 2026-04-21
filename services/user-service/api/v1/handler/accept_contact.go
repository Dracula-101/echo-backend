package handler

import (
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/internal/domain"
	userErrors "user-service/internal/errors"

	"github.com/google/uuid"
)

func (h *UserHandler) AcceptContact(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestId := handler.GetRequestID()
	correlationId := handler.GetCorrelationID()

	h.log.Info("Received request to accept contact",
		logger.String("request_id", requestId),
		logger.String("correlation_id", correlationId),
	)

	contactIdStr := handler.PathParam("contact_id")
	contactId, err := uuid.Parse(contactIdStr)
	if err != nil {
		h.log.Info("Invalid contact ID format for accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invalid contact ID format", nil)
		return
	}

	userId, ok := request.GetUserIDUUIDFromContext(ctx)
	if !ok {
		h.log.Info("User ID not found in context for accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	contact, err := h.contactService.CheckContactById(ctx, userId.String(), contactId.String())
	if err != nil {
		h.log.Error("Failed to check contact existence for accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to accept contact", nil)
		return
	}

	if contact == nil {
		h.log.Info("Contact not found for accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.NotFoundError(ctx, handler.Request(), handler.Writer(), "Contact not found")
		return
	}

	switch contact.Status {
	case domain.ContactStatusActive:
		h.log.Info("Contact is already active for accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact is already active", nil)
		return
	case domain.ContactStatusBlocked:
		h.log.Info("Contact is blocked for accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact is blocked", nil)
		return
	case domain.ContactStatusPending:
		// continue with rejection
		break
	case domain.ContactStatusDeleted:
		h.log.Info("Contact is deleted for accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact is deleted", nil)
		return
	default:
		h.log.Info("Contact is in invalid status for rejecting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.String("contact_status", string(contact.Status)),
			logger.String("expected_status", string(domain.ContactStatusPending)),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact is in invalid status for rejection", nil)
		return
	}

	counterpartyUserID := contact.UserID

	ok, cacheErr := h.cache.AcquireContactLock(ctx, userId.String(), counterpartyUserID)
	if cacheErr != nil {
		h.log.Error("Failed to acquire contact lock from cache for accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.Error(cacheErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to accept contact", nil)
		return
	}
	if !ok {
		h.log.Info("Contact lock is already held for accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.ConflictError(ctx, handler.Request(), handler.Writer(), "Contact is being processed. Please try again later.", nil)
		return
	}
	defer func() {
		releaseErr := h.cache.ReleaseContactLock(ctx, userId.String(), counterpartyUserID)
		if releaseErr != nil {
			h.log.Error("Failed to release contact lock in cache for accepting contact",
				logger.String("request_id", requestId),
				logger.String("correlation_id", correlationId),
				logger.String("contact_id", contactIdStr),
				logger.Error(releaseErr),
			)
		}
	}()

	err = h.contactService.AcceptContact(ctx, userId.String(), contactId.String())
	if err != nil {
		h.log.Error("Failed to accept contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.Error(err),
		)
		if pkgErrors.GetCode(err) == userErrors.ErrCodeInvalidRequest {
			response.ConflictError(ctx, handler.Request(), handler.Writer(), "Contact request is no longer pending", nil)
			return
		}
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to accept contact", nil)
		return
	}

	cacheErr = h.cache.AddContactID(ctx, userId.String(), counterpartyUserID)
	if cacheErr != nil {
		h.log.Error("Failed to add contact ID to cache after accepting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.Error(cacheErr),
		)
	}

	h.log.Info("Contact accepted successfully",
		logger.String("request_id", requestId),
		logger.String("correlation_id", correlationId),
		logger.String("contact_id", contactIdStr),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusAccepted, "Contact accepted successfully", nil)
}
