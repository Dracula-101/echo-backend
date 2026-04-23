package handler

import (
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *UserHandler) RemoveContact(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestId := handler.GetRequestID()
	correlationId := handler.GetCorrelationID()

	h.log.Info("Received request to remove contact",
		logger.String("request_id", requestId),
		logger.String("correlation_id", correlationId),
	)

	contactIdStr := handler.PathParam("contact_id")
	contact, err := uuid.Parse(contactIdStr)
	if err != nil {
		h.log.Info("Invalid contact ID format for removing contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact not found", nil)
		return
	}

	userId, ok := request.GetUserIDUUIDFromContext(ctx)
	if !ok {
		h.log.Info("User ID not found in context for removing contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	err = h.contactService.RemoveContact(ctx, userId.String(), contact.String())
	if err != nil {
		h.log.Error("Failed to remove contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_id", contactIdStr),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to remove contact", nil)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Contact removed successfully", nil)
}
