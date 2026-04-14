package handler

import (
	"auth-service/api/v1/dto"
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

	dto := dto.NewGetSessionsRequest()
	if ok := handler.ParseValidateAndSend(dto); !ok {
		h.log.Warn("Get sessions request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("correlation_id", correlationID),
		)
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid request payload", nil)
		return
	}

	userId, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Warn("User ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	sessions, err := h.sessionService.GetAllSessions(ctx, userId, dto.Limit, dto.Offset)
	if err != nil {
		h.log.Warn("Get sessions failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.Error(err),
		)
		response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), "Failed to get sessions", err)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), response.StatusOK, "Sessions retrieved successfully", sessions)
}
