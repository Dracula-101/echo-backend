package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// BookmarkMessage bookmarks a message for the current user
func (h *MessageHandler) BookmarkMessage(handler *req.RequestHandler) {
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

	request := dto.NewCreateBookmarkRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	h.log.Debug("Bookmarking message",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
	)

	svcErr := h.bookmarkService.CreateBookmark(ctx, msgUUID, uuid.MustParse(userID), request.Notes)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Message bookmarked successfully", nil)
}

// RemoveBookmark removes a bookmark from a message
func (h *MessageHandler) RemoveBookmark(handler *req.RequestHandler) {
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

	svcErr := h.bookmarkService.DeleteBookmark(ctx, msgUUID, uuid.MustParse(userID))
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Bookmark removed successfully", nil)
}

// GetBookmarks retrieves the current user's bookmarked messages
func (h *MessageHandler) GetBookmarks(handler *req.RequestHandler) {
	ctx := handler.Context()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	limit, err := handler.QueryParamInt("limit", 50)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be an integer", nil)
		return
	}
	if limit < 1 || limit > 100 {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be between 1 and 100", nil)
		return
	}

	offset, err := handler.QueryParamInt("offset", 0)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "offset must be an integer", nil)
		return
	}

	bookmarks, totalCount, svcErr := h.bookmarkService.GetBookmarks(ctx, uuid.MustParse(userID), limit, offset)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Bookmarks retrieved",
		map[string]interface{}{
			"bookmarks":   bookmarks,
			"total_count": totalCount,
			"limit":       limit,
			"offset":      offset,
		},
	)
}
