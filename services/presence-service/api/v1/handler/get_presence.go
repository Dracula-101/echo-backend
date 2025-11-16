package handler

import (
	"net/http"
	"presence-service/internal/errors"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *PresenceHandler) GetPresence(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Get presence request received",
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

	requesterIDStr := r.Header.Get("X-User-ID")
	requesterID, err := uuid.Parse(requesterIDStr)
	if err != nil {
		response.UnauthorizedError(r.Context(), r, w, "Missing or invalid requester ID", err)
		return
	}

	presence, svcErr := h.service.GetPresence(r.Context(), userId, requesterID)
	if svcErr != nil {
		if appErr, ok := svcErr.(pkgErrors.AppError); ok {
			h.log.Error("Failed to get presence",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to get presence", logger.Error(svcErr))
		}
		response.InternalServerError(r.Context(), r, w, "Failed to get presence", svcErr)
		return
	}

	response.JSONWithContext(r.Context(), r, w, http.StatusOK, presence)
}
