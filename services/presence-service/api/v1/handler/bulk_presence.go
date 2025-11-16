package handler

import (
	"encoding/json"
	"net/http"
	"presence-service/internal/errors"
	"presence-service/internal/model"

	"github.com/google/uuid"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

func (h *PresenceHandler) GetBulkPresence(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Get bulk presence request received",
		logger.String("service", errors.ServiceName),
		logger.String("request_id", requestID),
	)

	requesterIDStr := r.Header.Get("X-User-ID")
	requesterID, err := uuid.Parse(requesterIDStr)
	if err != nil {
		response.UnauthorizedError(r.Context(), r, w, "Missing or invalid requester ID", err)
		return
	}

	var req model.BulkPresenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequestError(r.Context(), r, w, "Invalid request body", err)
		return
	}

	if len(req.UserIDs) == 0 {
		response.BadRequestError(r.Context(), r, w, "User IDs required", nil)
		return
	}

	presences, svcErr := h.service.GetBulkPresence(r.Context(), req.UserIDs, requesterID)
	if svcErr != nil {
		if appErr, ok := svcErr.(pkgErrors.AppError); ok {
			h.log.Error("Failed to get bulk presence",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to get bulk presence", logger.Error(svcErr))
		}
		response.InternalServerError(r.Context(), r, w, "Failed to get bulk presence", svcErr)
		return
	}

	resp := model.BulkPresenceResponse{
		Presences: make(map[uuid.UUID]model.UserPresence),
	}

	for userID, presence := range presences {
		resp.Presences[userID] = *presence
	}

	response.JSONWithContext(r.Context(), r, w, http.StatusOK, resp)
}
