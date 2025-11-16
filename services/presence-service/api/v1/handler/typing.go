package handler

import (
	"encoding/json"
	"net/http"
	"presence-service/internal/model"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func (h *PresenceHandler) SetTypingIndicator(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	requestID := handler.GetRequestID()

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		response.BadRequestError(r.Context(), r, w, "Missing user ID", nil)
		return
	}

	userId, ok := request.GetUserIDUUIDFromContext(r.Context())
	if !ok {
		h.log.Warn("User ID missing in context for getting presence",
			logger.String("request_id", requestID),
		)
		response.BadRequestError(r.Context(), r, w, "User ID missing in context", nil)
		return
	}

	var req model.TypingIndicator
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequestError(r.Context(), r, w, "Invalid request body", err)
		return
	}

	req.UserID = userId
	req.DeviceID = r.Header.Get("X-Device-ID")

	if svcErr := h.service.SetTypingIndicator(r.Context(), &req); svcErr != nil {
		if appErr, ok := svcErr.(pkgErrors.AppError); ok {
			h.log.Error("Failed to set typing indicator",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to set typing indicator", logger.Error(svcErr))
		}
		response.InternalServerError(r.Context(), r, w, "Failed to set typing indicator", svcErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PresenceHandler) GetTypingIndicators(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationIDStr := vars["conversation_id"]

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		response.BadRequestError(r.Context(), r, w, "Invalid conversation ID", err)
		return
	}

	indicators, svcErr := h.service.GetTypingIndicators(r.Context(), conversationID)
	if svcErr != nil {
		if appErr, ok := svcErr.(pkgErrors.AppError); ok {
			h.log.Error("Failed to get typing indicators",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to get typing indicators", logger.Error(svcErr))
		}
		response.InternalServerError(r.Context(), r, w, "Failed to get typing indicators", svcErr)
		return
	}

	response.JSONWithContext(r.Context(), r, w, http.StatusOK, map[string]any{
		"typing_users": indicators,
		"count":        len(indicators),
	})
}
