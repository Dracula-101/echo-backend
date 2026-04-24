package handler

import (
	"strconv"
	"strings"
	"user-service/api/v1/dto"
	"user-service/internal/service/contact"

	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
)

func (h *UserHandler) ListContacts(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestId := handler.GetRequestID()
	correlationId := handler.GetCorrelationID()

	h.log.Info("ListContacts called",
		logger.String("requestId", requestId),
		logger.String("correlationId", correlationId),
	)

	userId, ok := request.GetUserIDUUIDFromContext(ctx)
	if !ok {
		h.log.Error("Failed to get user ID from context",
			logger.String("requestId", requestId),
			logger.String("correlationId", correlationId),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	status := strings.TrimSpace(handler.QueryParamDefault("status", "active"))
	relationship := strings.TrimSpace(handler.QueryParam("relationship"))
	groupID := strings.TrimSpace(handler.QueryParam("group_id"))
	search := strings.TrimSpace(handler.QueryParam("search"))

	pageStr := handler.QueryParamDefault("page", "1")
	limitStr := handler.QueryParamDefault("limit", "50")

	page, parseErr := strconv.Atoi(pageStr)
	if parseErr != nil || page <= 0 {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "page must be a positive integer", nil)
		return
	}

	limit, parseErr := strconv.Atoi(limitStr)
	if parseErr != nil || limit <= 0 {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be a positive integer", nil)
		return
	}

	params := contact.GetContactsParams{
		Status:       status,
		Relationship: relationship,
		Page:         page,
		Limit:        limit,
	}

	if groupID != "" {
		params.GroupID = &groupID
	}
	if search != "" {
		params.Search = &search
	}

	result, err := h.contactService.GetContacts(ctx, userId.String(), params)
	if err != nil {
		h.log.Error("Failed to list contacts",
			logger.String("requestId", requestId),
			logger.String("correlationId", correlationId),
			logger.String("user_id", userId.String()),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to list contacts", err)
		return
	}

	resp := dto.NewListContactsResponse(result.Contacts, result.Total, result.Page, result.Limit, result.UnreadRequestsCount)
	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Contacts retrieved successfully", resp)
}
