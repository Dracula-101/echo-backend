package handler

import (
	"net/http"
	"presence-service/internal/errors"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

func (h *PresenceHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Debug("Heartbeat received",
		logger.String("service", errors.ServiceName),
		logger.String("request_id", requestID),
	)

	userId, ok := request.GetUserIDUUIDFromContext(r.Context())
	if !ok {
		h.log.Warn("User ID missing in context for getting presence",
			logger.String("request_id", requestID),
		)
		response.BadRequestError(r.Context(), r, w, "User ID missing in context", nil)
		return
	}

	deviceID := r.Header.Get("X-Device-ID")
	if deviceID == "" {
		response.BadRequestError(r.Context(), r, w, "Missing device ID", nil)
		return
	}

	if svcErr := h.service.Heartbeat(r.Context(), userId, deviceID); svcErr != nil {
		if appErr, ok := svcErr.(pkgErrors.AppError); ok {
			h.log.Error("Failed to process heartbeat",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to process heartbeat", logger.Error(svcErr))
		}
		response.InternalServerError(r.Context(), r, w, "Failed to process heartbeat", svcErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
