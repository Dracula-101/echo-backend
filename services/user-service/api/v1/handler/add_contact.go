package handler

import (
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"time"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
)

func (h *UserHandler) AddContact(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestId := handler.GetRequestID()
	correlationId := handler.GetCorrelationID()

	h.log.Info("Received request to add contact",
		logger.String("request_id", requestId),
		logger.String("correlation_id", correlationId),
	)

	addContactDto := dto.NewAddContactRequest()
	if ok := handler.ParseValidateAndSend(addContactDto); !ok {
		h.log.Info("Invalid request payload for adding contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		return
	}

	userId, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Info("User ID not found in context for adding contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}
	if addContactDto.IdentifierType == dto.IdentifierTypePhone {
		_, err := utils.ValidatePhoneNumberE164(addContactDto.Identifier)
		if err != nil {
			h.log.Info("Invalid phone number format for adding contact",
				logger.String("request_id", requestId),
				logger.String("correlation_id", correlationId),
				logger.String("phone_number", addContactDto.Identifier),
			)
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invalid phone number format", nil)
			return
		}
	}
	if addContactDto.Identifier == userId {
		h.log.Info("User attempted to add themselves as a contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Cannot add yourself as a contact", nil)
		return
	}

	ids, cacheErr := h.cache.GetBlockedIDs(ctx, userId)
	if cacheErr != nil {
		h.log.Error("Failed to retrieve blocked IDs from cache",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.Error(cacheErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process contact request", nil)
		return
	} else if ids != nil {
		for _, blockedId := range *ids {
			if blockedId == addContactDto.Identifier {
				h.log.Info("Cannot add contact because the user is blocked",
					logger.String("request_id", requestId),
					logger.String("correlation_id", correlationId),
					logger.String("blocked_user_id", addContactDto.Identifier),
				)
				response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Cannot add contact because the user is blocked", nil)
				return
			}
		}
	}

	ids, cacheErr = h.cache.GetBlockedIDs(ctx, addContactDto.Identifier)
	if cacheErr != nil {
		h.log.Error("Failed to retrieve blocked IDs from cache for contact identifier",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.String("contact_identifier", addContactDto.Identifier),
			logger.Error(cacheErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process contact request", nil)
		return
	} else if ids != nil {
		for _, blockedId := range *ids {
			if blockedId == userId {
				h.log.Info("Cannot add contact because the user has blocked you",
					logger.String("request_id", requestId),
					logger.String("correlation_id", correlationId),
					logger.String("blocked_by_user_id", addContactDto.Identifier),
				)
				response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Cannot add contact because the user has blocked you", nil)
				return
			}
		}
	}

	contact, err := h.contactService.CheckContactExists(ctx, userId, addContactDto.Identifier)
	if err != nil {
		h.log.Error("Error checking if contact exists",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to check existing contacts", nil)
		return
	}
	if contact != nil {
		switch contact.Status {
		case domain.ContactStatusPending:
			h.log.Info("Contact request already pending",
				logger.String("request_id", requestId),
				logger.String("correlation_id", correlationId),
			)
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact request already pending", nil)
			return
		case domain.ContactStatusActive:
			h.log.Info("Contact already exists",
				logger.String("request_id", requestId),
				logger.String("correlation_id", correlationId),
			)
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact already exists", nil)
			return
		case domain.ContactStatusDeleted:
			// 7 days have passed since deletion, allow re-adding contact
			if contact.UpdatedAt != nil && time.Since(*contact.UpdatedAt) < 7*24*time.Hour {
				h.log.Info("Contact was recently deleted, cannot re-add yet",
					logger.String("request_id", requestId),
					logger.String("correlation_id", correlationId),
				)
				response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Contact was recently deleted, please wait before re-adding", nil)
				return
			}
		}
	}

	ok, cacheEr := h.cache.AcquireContactLock(ctx, userId, addContactDto.Identifier)
	if cacheEr != nil {
		h.log.Error("Failed to acquire contact lock from cache",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.Error(cacheEr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process contact request", nil)
		return
	}
	if !ok {
		h.log.Info("Another contact request is being processed for the same user pair",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Another contact request is being processed for the same user pair, please try again later", nil)
		return
	}
	defer func() {
		err := h.cache.ReleaseContactLock(ctx, userId, addContactDto.Identifier)
		if err != nil {
			h.log.Error("Failed to release contact lock in cache",
				logger.String("request_id", requestId),
				logger.String("correlation_id", correlationId),
				logger.Error(err),
			)
		}
	}()
	trimmedMessage := utils.TrimString(utils.SafeString(addContactDto.Message), 200)
	err = h.contactService.AddContact(ctx, userId, addContactDto.Identifier, addContactDto.IdentifierType, trimmedMessage, domain.ContactSource(addContactDto.Source))
	if err != nil {
		h.log.Error("Failed to add contact",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to add contact", nil)
		return
	}

	h.log.Info("Contact added successfully",
		logger.String("request_id", requestId),
		logger.String("correlation_id", correlationId),
	)
}
