package handler

import (
	"encoding/json"
	"net/http"
	"presence-service/internal/errors"
	"presence-service/internal/model"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

func (h *PresenceHandler) UpdatePresence(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Update presence request received",
		logger.String("service", errors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	userId, ok := request.GetUserIDUUIDFromContext(r.Context())
	if !ok {
		h.log.Warn("User ID missing in context for updating presence",
			logger.String("request_id", requestID),
		)
		response.BadRequestError(r.Context(), r, w, "User ID missing in context", nil)
		return
	}

	var req model.PresenceUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Update presence parsing failed",
			logger.String("service", errors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.BadRequestError(r.Context(), r, w, "Invalid request body", err)
		return
	}

	req.UserID = userId

	presence, svcErr := h.service.UpdatePresence(r.Context(), &req)
	if svcErr != nil {
		if appErr, ok := svcErr.(pkgErrors.AppError); ok {
			h.log.Error("Failed to update presence",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.String("correlation_id", appErr.CorrelationID()),
				logger.Any("error_details", appErr.Details()),
				logger.Any("stack_trace", appErr.StackTrace()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to update presence", logger.Error(svcErr))
		}
		response.InternalServerError(r.Context(), r, w, "Failed to update presence", svcErr)
		return
	}

	response.JSONWithContext(r.Context(), r, w, http.StatusOK, presence)
}
