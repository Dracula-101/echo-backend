package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// ReportMessage reports a message for moderation
func (h *MessageHandler) ReportMessage(handler *req.RequestHandler) {
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
	msgUUID, err := uuid.Parse(messageID)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Message ID must be a valid UUID", nil)
		return
	}

	request := dto.NewReportMessageRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	h.log.Debug("Reporting message",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
		logger.String("report_type", request.ReportType),
	)

	svcErr := h.reportService.ReportMessage(ctx, msgUUID, uuid.MustParse(userID), request.ReportType, request.Description)
	if svcErr != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Message reported successfully", nil)
}
