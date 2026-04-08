package handler

import (
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) DeleteSession(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Delete session request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	sessionID := handler.PathParam("id")
	sessionErr := h.sessionService.DeleteSessionByID(ctx, sessionID)
	if sessionErr != nil {
		h.log.Warn("Delete session failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("session_id", sessionID),
			logger.Error(sessionErr),
		)
		response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), "Failed to delete session", sessionErr)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), response.StatusOK, "Session deleted successfully", nil)
}
