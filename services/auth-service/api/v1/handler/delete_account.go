package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// Delete account flow at a glance:
//
//	[1] Context extraction — get user ID from context.
//	[2] Request helper — parse & validate JSON (password confirmation).
//	[3] AuthService.DeleteAccount — verify password, soft-delete user, revoke sessions.
//	[4] Response helper — return 200 JSON body on success.
//	[5] Failure handling — log and return structured error payloads on failure.
func (h *AuthHandler) DeleteAccount(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Delete account request received",
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

	deleteRequest := dto.NewDeleteAccountRequest()
	if !handler.ParseValidateAndSend(deleteRequest) {
		h.log.Warn("Delete account request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	authErr := h.authService.DeleteAccount(ctx, userID, deleteRequest.Password, handler.GetClientIP(), handler.GetUserAgent())
	if authErr != nil {
		h.log.Warn("Delete account failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(authErr.Error),
		)

		switch authErr.Code {
		case authErrors.CodeInvalidCredentials:
			response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Invalid password", nil)
			return
		case authErrors.CodeUserNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), "User not found")
			return
		case authErrors.CodeAccountAlreadyDeleted:
			response.ConflictError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
			return
		}

		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to delete account", authErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Account deleted successfully", nil)
}
