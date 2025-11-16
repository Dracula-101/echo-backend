package handler

import (
	"net/http"
	"presence-service/api/v1/dto"
	"presence-service/internal/model"
	"time"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// BroadcastEvent handles event broadcasting requests from other services
// This is the primary endpoint that services like message-service, call-service, etc.
// use to send real-time events to connected users
func (h *PresenceHandler) BroadcastEvent(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Broadcast event request received",
		logger.String("service", "presence-service"),
		logger.String("request_id", requestID),
		logger.String("source_service", r.Header.Get("X-Source-Service")),
	)

	dtoReq := dto.NewBroadcastEventRequest()
	if ok := handler.ParseValidateAndSend(dtoReq); !ok {
		h.log.Warn("Broadcast request validation failed",
			logger.String("request_id", requestID),
		)
		return
	}

	// Convert DTO to model
	req, err := dtoReq.ToModel()
	if err != nil {
		response.BadRequestError(r.Context(), r, w, "Invalid UUID format", err)
		return
	}

	// Create real-time event
	event := &model.RealtimeEvent{
		ID:         uuid.New(),
		Type:       req.EventType,
		Category:   getEventCategory(req.EventType),
		Timestamp:  time.Now(),
		Recipients: req.Recipients,
		Sender:     req.Sender,
		Payload:    req.Payload,
		Priority:   req.Priority,
		TTL:        req.TTL,
	}

	// Broadcast via service
	if err := h.service.BroadcastEvent(r.Context(), event); err != nil {
		if appErr, ok := err.(pkgErrors.AppError); ok {
			h.log.Error("Failed to broadcast event",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to broadcast event", logger.Error(err))
		}
		response.InternalServerError(r.Context(), r, w, "Failed to broadcast event", err)
		return
	}

	h.log.Debug("Event broadcasted successfully",
		logger.String("event_type", string(req.EventType)),
		logger.Int("recipients", len(req.Recipients)),
	)

	response.JSON(w, http.StatusAccepted, map[string]interface{}{
		"success":    true,
		"event_id":   event.ID,
		"recipients": len(req.Recipients),
		"timestamp":  event.Timestamp,
	})
}

// BroadcastToUser broadcasts an event to a specific user (all their devices)
func (h *PresenceHandler) BroadcastToUser(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Broadcast to user request received",
		logger.String("request_id", requestID),
	)

	// Get user ID
	userId, isPresent := request.GetUserIDUUIDFromContext(r.Context())
	if !isPresent {
		h.log.Warn("User ID missing in context for broadcast to user",
			logger.String("request_id", requestID),
		)
		response.BadRequestError(r.Context(), r, w, "User ID missing in context", nil)
		return
	}

	// Parse broadcast request
	dtoReq := dto.NewBroadcastEventRequest()
	if ok := handler.ParseValidateAndSend(dtoReq); !ok {
		return
	}

	// Convert DTO to model
	req, err := dtoReq.ToModel()
	if err != nil {
		response.BadRequestError(r.Context(), r, w, "Invalid UUID format", err)
		return
	}

	// Override recipients with specified user
	req.Recipients = []uuid.UUID{userId}

	// Create real-time event
	event := &model.RealtimeEvent{
		ID:         uuid.New(),
		Type:       req.EventType,
		Category:   getEventCategory(req.EventType),
		Timestamp:  time.Now(),
		Recipients: req.Recipients,
		Sender:     req.Sender,
		Payload:    req.Payload,
		Priority:   req.Priority,
		TTL:        req.TTL,
	}

	// Broadcast via service
	if err := h.service.BroadcastEvent(r.Context(), event); err != nil {
		response.InternalServerError(r.Context(), r, w, "Failed to broadcast event", err)
		return
	}

	response.JSON(w, http.StatusAccepted, map[string]interface{}{
		"success":   true,
		"event_id":  event.ID,
		"user_id":   userId,
		"timestamp": event.Timestamp,
	})
}

// getEventCategory extracts the category from event type
func getEventCategory(eventType model.EventType) model.EventCategory {
	switch eventType {
	case model.EventPresenceOnline, model.EventPresenceOffline, model.EventPresenceAway,
		model.EventPresenceBusy, model.EventPresenceInvisible, model.EventPresenceUpdate:
		return model.CategoryPresence

	case model.EventMessageNew, model.EventMessageDelivered, model.EventMessageRead,
		model.EventMessageEdited, model.EventMessageDeleted:
		return model.CategoryMessaging

	case model.EventTypingStart, model.EventTypingStop:
		return model.CategoryTyping

	case model.EventCallIncoming, model.EventCallAccepted, model.EventCallRejected,
		model.EventCallEnded, model.EventCallMissed:
		return model.CategoryCall

	case model.EventNotificationNew, model.EventNotificationRead:
		return model.CategoryNotification

	case model.EventUserProfileUpdated, model.EventUserStatusUpdated,
		model.EventUserBlocked, model.EventUserUnblocked:
		return model.CategoryUser

	case model.EventSystemMaintenance, model.EventSystemAnnouncement:
		return model.CategorySystem

	default:
		return model.CategorySystem
	}
}
