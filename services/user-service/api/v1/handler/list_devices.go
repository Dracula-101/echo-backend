package handler

import (
	"net/http"

	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
)

func (h *UserHandler) ListDevices(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Received request to list devices",
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	currentDeviceID, _ := handler.GetDeviceIDFromHeader()

	devs, err := h.deviceRepo.GetDevices(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get devices",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to retrieve devices", nil)
		return
	}

	items := make([]*dto.DeviceResponse, len(devs))
	for i, d := range devs {
		items[i] = dto.NewDeviceResponse(d, currentDeviceID)
	}

	h.log.Info("Devices listed successfully",
		logger.String("request_id", requestID),
		logger.String("user_id", userID),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), http.StatusOK, "Devices retrieved successfully", map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}
