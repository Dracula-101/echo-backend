package handler

import (
	"net/http"
	"presence-service/internal/errors"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

func (h *PresenceHandler) GetActiveDevices(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Get active devices request received",
		logger.String("service", errors.ServiceName),
		logger.String("request_id", requestID),
	)

	userID, ok := request.GetUserIDUUIDFromContext(r.Context())
	if !ok {
		h.log.Warn("User ID missing in context for getting active devices",
			logger.String("request_id", requestID),
		)
		response.BadRequestError(r.Context(), r, w, "User ID missing in context", nil)
		return
	}

	devices, svcErr := h.service.GetActiveDevices(r.Context(), userID)
	if svcErr != nil {
		if appErr, ok := svcErr.(pkgErrors.AppError); ok {
			h.log.Error("Failed to get active devices",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to get active devices", logger.Error(svcErr))
		}
		response.InternalServerError(r.Context(), r, w, "Failed to get active devices", svcErr)
		return
	}

	response.JSONWithContext(r.Context(), r, w, http.StatusOK, map[string]any{
		"devices": devices,
		"count":   len(devices),
	})
}
