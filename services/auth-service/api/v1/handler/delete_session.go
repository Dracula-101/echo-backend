package handler

import (
	authErrors "auth-service/internal/error"
	repository "auth-service/internal/repo"
	models "shared/pkg/database/postgres/models"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// Delete session flow at a glance:
//
//	[1] Context extraction — get user ID from context.
//	[2] Path param — extract session ID from URL path.
//	[3] Ownership check — verify the session belongs to the authenticated user.
//	[4] SessionService.DeleteSessionByID — delete the session.
//	[5] Response helper — return 200 JSON body on success.
//	[6] Failure handling — log and return structured error payloads on failure.
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

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Warn("User ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	sessionID := handler.PathParam("id")
	if sessionID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Session ID is required", nil)
		return
	}

	session, getErr := h.sessionService.GetSessionByID(ctx, sessionID)
	if getErr != nil {
		h.log.Warn("Failed to get session for ownership check",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("session_id", sessionID),
			logger.Error(getErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to delete session", getErr)
		return
	}
	if session == nil {
		response.NotFoundError(ctx, handler.Request(), handler.Writer(), "Session not found")
		return
	}
	if session.UserID != userID {
		h.log.Warn("Session ownership mismatch",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("session_id", sessionID),
			logger.String("session_owner", session.UserID),
			logger.String("requester", userID),
		)
		response.ForbiddenError(ctx, handler.Request(), handler.Writer(), "You do not own this session", nil)
		return
	}

	sessionErr := h.sessionService.DeleteSessionByID(ctx, sessionID)
	if sessionErr != nil {
		h.log.Warn("Delete session failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("session_id", sessionID),
			logger.Error(sessionErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to delete session", sessionErr)
		return
	}

	h.log.Info("Session deleted successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("session_id", sessionID),
	)
	h.securityEventRepo.LogEvent(ctx, repository.SecurityEventInput{
		UserID:        userID,
		EventType:     models.SecurityEventSessionRevoked,
		IPAddress:     handler.GetClientIP(),
		UserAgent:     handler.GetUserAgent(),
		EventCategory: string(models.SecurityEventSessionRevoked),
		Severity:      models.SecuritySeverityMedium,
		Status:        "success",
		Description:   "User deleted their session successfully",
	})
	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Session deleted successfully", nil)
}
