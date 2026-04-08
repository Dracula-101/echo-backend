package handler

import (
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

	userId, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Warn("User ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	sessions, err := h.sessionService.GetAllSessions(ctx, userId)
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
