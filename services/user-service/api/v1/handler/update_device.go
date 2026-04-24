package handler

import (
	"net/http"

	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
)

func (h *UserHandler) UpdateDevice(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Received request to update device push token",
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	deviceID := handler.PathParam("device_id")

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	body := dto.NewUpdatePushTokenRequest()
	if ok := handler.ParseValidateAndSend(body); !ok {
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
	if device == nil || !device.IsActive {
		response.NotFoundError(ctx, handler.Request(), handler.Writer(), "device")
		return
	}

	if err := h.deviceRepo.UpdatePushToken(ctx, device.ID, body.FCMToken, body.APNSToken, body.PushEnabled); err != nil {
		h.log.Error("Failed to update push token",
			logger.String("request_id", requestID),
			logger.String("device_id", deviceID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to update push token", nil)
		return
	}

	if body.FCMToken != nil || body.APNSToken != nil {
		if err := h.deviceRepo.UpdateAuthSessionPushToken(ctx, userID, deviceID, body.FCMToken, body.APNSToken); err != nil {
			h.log.Error("Failed to update auth session push token",
				logger.String("request_id", requestID),
				logger.String("device_id", deviceID),
				logger.Error(err),
			)
		}
	}

	h.log.Info("Device push token updated successfully",
		logger.String("request_id", requestID),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)

	handler.Writer().WriteHeader(http.StatusNoContent)
}
