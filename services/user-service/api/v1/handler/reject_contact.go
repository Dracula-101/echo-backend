package handler

import (
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/internal/domain"

	"github.com/google/uuid"
)

func (h *UserHandler) RejectContact(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestId := handler.GetRequestID()
	correlationId := handler.GetCorrelationID()

	h.log.Info("Received request to reject contact",
		logger.String("request_id", requestId),
		logger.String("correlation_id", correlationId),
	)

	contactIdStr := handler.PathParam("contact_id")
	contactId, err := uuid.Parse(contactIdStr)
	if err != nil {
		h.log.Info("Invalid contact ID format for rejecting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invalid contact ID format", nil)
		return
	}

	userId, ok := request.GetUserIDUUIDFromContext(ctx)
	if !ok {
		h.log.Info("User ID not found in context for rejecting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	contact, err := h.contactService.CheckContactById(ctx, userId.String(), contactId.String())
	if err != nil {
		h.log.Error("Failed to check contact existence for rejecting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to reject contact", nil)
		return
	}

	if contact == nil {
		h.log.Info("Contact not found for rejecting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.NotFoundError(ctx, handler.Request(), handler.Writer(), "Contact not found")
		return
	}

	switch contact.Status {
	case domain.ContactStatusDeleted:
		h.log.Info("Contact is already deleted for rejecting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.String("contact_status", string(contact.Status)),
			logger.String("expected_status", string(domain.ContactStatusDeleted)),
			logger.Error(err),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact is already deleted", nil)
		return
	case domain.ContactStatusActive:
		h.log.Info("Contact is active, cannot reject active contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.String("contact_status", string(contact.Status)),
			logger.String("expected_status", string(domain.ContactStatusPending)),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Cannot reject an active contact. Please delete the contact instead.", nil)
		return
	case domain.ContactStatusBlocked:
		h.log.Info("Contact is blocked, cannot reject blocked contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.String("contact_status", string(contact.Status)),
			logger.String("expected_status", string(domain.ContactStatusPending)),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Cannot reject a blocked contact. Please unblock the contact instead.", nil)
		return
	case domain.ContactStatusPending:
		// continue with rejection
		break
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

	ok, cacheErr := h.cache.AcquireContactLock(ctx, userId.String(), contact.ContactUserID)
	if cacheErr != nil {
		h.log.Error("Failed to acquire contact lock for rejecting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.Error(cacheErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to reject contact", nil)
		return
	}
	if !ok {
		h.log.Info("Contact lock is already held, cannot reject contact at the moment",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.ConflictError(ctx, handler.Request(), handler.Writer(), "Contact is being processed. Please try again later.", nil)
		return
	}
	defer func() {
		releaseErr := h.cache.ReleaseContactLock(ctx, userId.String(), contact.ContactUserID)
		if releaseErr != nil {
			h.log.Error("Failed to release contact lock for rejecting contact",
				logger.String("request_id", requestId),
				logger.String("correlation_id", correlationId),
				logger.String("contact_id", contactIdStr),
				logger.Error(releaseErr),
			)
		}
	}()

	err = h.contactService.RejectContact(ctx, userId.String(), contactId.String())
	if err != nil {
		h.log.Error("Failed to reject contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to reject contact", nil)
		return
	}

	cacheErr = h.cache.AddBlockedID(ctx, userId.String(), contact.ContactUserID)
	if cacheErr != nil {
		h.log.Error("Failed to add blocked ID to cache after rejecting contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.Error(cacheErr),
		)
		// not returning error response since the main operation of rejecting contact has succeeded
	}

	h.log.Info("Contact rejected successfully",
		logger.String("request_id", requestId),
		logger.String("correlation_id", correlationId),
		logger.String("contact_id", contactIdStr),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Contact rejected successfully", nil)
}
