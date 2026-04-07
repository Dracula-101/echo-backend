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
//		[1] Context extraction — get user ID from context.
//		[2] AuthService.DeleteAccount — delete user account and associated data.
//		[3] Response helper — return 200 JSON body on success.
//	 [4] Failure handling — log and return structured error payloads on failure.
//
// Note: This endpoint requires authentication and should be protected by middleware that populates the user ID in the context.
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

	userID, noFound := req.GetUserIDFromContext(ctx)
	if noFound {
		h.log.Warn("User ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	passwordChangeRequest := dto.NewChangePasswordRequest()
	if !handler.ParseValidateAndSend(passwordChangeRequest) {
		h.log.Warn("Delete account request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	authErr := h.authService.DeleteAccount(ctx, userID, passwordChangeRequest.CurrentPassword)
	if authErr != nil {
		h.log.Warn("Delete account failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(authErr.Error),
		)
		response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), "Failed to delete account", authErr.Error)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), response.StatusOK, "Account deleted successfully", nil)
}
