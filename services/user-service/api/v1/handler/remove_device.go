package handler

import (
	"net/http"

	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	userErrors "user-service/internal/errors"
)

func (h *UserHandler) RemoveDevice(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Received request to remove device",
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	deviceID := handler.PathParam("device_id")

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	currentDeviceID, _ := handler.GetDeviceIDFromHeader()
	if currentDeviceID != "" && deviceID == currentDeviceID {
		h.log.Info("Attempted to remove current device",
			logger.String("request_id", requestID),
			logger.String("device_id", deviceID),
		)
		response.Error().
			WithContext(ctx).
			WithRequest(handler.Request()).
			WithMessage("Cannot remove the current device — use logout instead").
			WithError(&response.ErrorDetails{
				Code:        userErrors.ErrCodeCannotRemoveCurrentDevice,
				Description: "Use the logout endpoint to remove your current device.",
			}).
			UnprocessableEntity(handler.Writer())
		return
	}

	device, err := h.deviceRepo.GetDevice(ctx, userID, deviceID)
	if err != nil {
		h.log.Error("Failed to look up device",
			logger.String("request_id", requestID),
			logger.String("device_id", deviceID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process request", nil)
		return
	}
	if device == nil {
		response.NotFoundError(ctx, handler.Request(), handler.Writer(), "device")
		return
	}

	if err := h.deviceRepo.DeactivateDevice(ctx, device.ID); err != nil {
		h.log.Error("Failed to deactivate device",
			logger.String("request_id", requestID),
			logger.String("device_id", deviceID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to remove device", nil)
		return
	}

	// TODO: publish user.device.removed to Kafka {user_id, device_id}
	// auth-service consumer: revoke auth.sessions for this user_id + device_id, delete Redis session keys
	// ws-service consumer: disconnect the session

	h.log.Info("Device removed successfully",
		logger.String("request_id", requestID),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)

	handler.Writer().WriteHeader(http.StatusNoContent)
}
