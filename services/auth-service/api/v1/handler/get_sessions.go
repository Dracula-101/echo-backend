package handler

import (
	"auth-service/api/v1/dto"
	"auth-service/internal/domain"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// Get sessions flow at a glance:
//
//	[1] Context extraction — get user ID from context.
//	[2] SessionService.GetAllSessions — fetch all sessions for the user.
//	[3] Response helper — return 200 with session list.
//	[4] Failure handling — log and return structured error payloads on failure.
func (h *AuthHandler) GetSessions(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Get sessions request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	params := dto.NewGetSessionsRequest()

	limit, err := handler.QueryParamInt("limit", params.Limit)
	if err != nil {
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid limit parameter", err)
		return
	}
	offset, err := handler.QueryParamInt("offset", params.Offset)
	if err != nil {
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid offset parameter", err)
		return
	}
	params.Limit = limit
	params.Offset = offset

	userId, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Warn("User ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	sessions, retLimit, retOffset, total, svcErr := h.sessionService.GetAllSessions(ctx, userId, params.Limit, params.Offset)
	if svcErr != nil {
		h.log.Warn("Get sessions failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.Error(svcErr),
		)
		response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), "Failed to get sessions", svcErr)
		return
	}

	sessionValues := make([]domain.Session, len(sessions))
	for i, s := range sessions {
		sessionValues[i] = *s
	}
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), response.StatusOK, "Sessions retrieved successfully", dto.GetSessionsResponse{
		Sessions: sessionValues,
		Limit:    retLimit,
		Offset:   retOffset,
		Total:    total,
	})
}
