package message

import (
	"time"

	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// DeliveryStatusResponse represents the delivery status summary response
type DeliveryStatusResponse struct {
	MessageID       string                `json:"message_id"`
	TotalRecipients int                   `json:"total_recipients"`
	DeliveredCount  int                   `json:"delivered_count"`
	ReadCount       int                   `json:"read_count"`
	PendingCount    int                   `json:"pending_count"`
	FailedCount     int                   `json:"failed_count"`
	ReadBy          []ReadByResponse      `json:"read_by,omitempty"`
	DeliveredBy     []DeliveredByResponse `json:"delivered_by,omitempty"`
}

type ReadByResponse struct {
	UserID string    `json:"user_id"`
	ReadAt time.Time `json:"read_at"`
}

type DeliveredByResponse struct {
	UserID      string    `json:"user_id"`
	DeliveredAt time.Time `json:"delivered_at"`
}

// GetDeliveryStatus retrieves the delivery status of a message
func (h *MessageHandler) GetDeliveryStatus(handler *req.RequestHandler) {
	ctx := handler.Context()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	messageID := handler.PathParam("id")
	if messageID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Message ID is required", nil)
		return
	}

	if _, err := uuid.Parse(messageID); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Message ID must be a valid UUID", nil)
		return
	}

	h.log.Debug("Getting delivery status",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
	)

	summary, err := h.messageService.GetDeliveryStatus(ctx, messageID, userID)
	if err != nil {
		switch err.Code {
		case msgError.CodeMessageNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), err.Message)
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		}
		return
	}

	// Build response
	resp := DeliveryStatusResponse{
		MessageID:       summary.MessageID.String(),
		TotalRecipients: summary.TotalRecipients,
		DeliveredCount:  summary.DeliveredCount,
		ReadCount:       summary.ReadCount,
		PendingCount:    summary.PendingCount,
		FailedCount:     summary.FailedCount,
	}
	for _, r := range summary.ReadBy {
		resp.ReadBy = append(resp.ReadBy, ReadByResponse{
			UserID: r.UserID.String(),
			ReadAt: r.ReadAt,
		})
	}
	for _, d := range summary.DeliveredBy {
		resp.DeliveredBy = append(resp.DeliveredBy, DeliveredByResponse{
			UserID:      d.UserID.String(),
			DeliveredAt: d.DeliveredAt,
		})
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Delivery status retrieved", resp)
}
