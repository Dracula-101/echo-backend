package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) GetUserConversations(handler *request.RequestHandler) {
	context := handler.Context()
	userIdInContext, _ := request.GetUserIDFromContext(context)
	userID, uuidErr := uuid.Parse(userIdInContext)
	if uuidErr != nil {
		h.log.Error("Invalid user ID in context",
			logger.String("user_id", userID.String()),
			logger.Error(uuidErr),
		)
		response.BadRequestError(context, handler.Request(), handler.Writer(), "Invalid user ID", uuidErr)
		return
	}

	conversations, err := h.conversationService.GetUserConversations(context, userID)
	if err != nil {
		h.log.Error("Failed to get user conversations",
			logger.String("user_id", userID.String()),
			logger.Error(err.Error),
		)
		response.InternalServerError(context, handler.Request(), handler.Writer(), "Failed to get user conversations", err.Error)
		return
	}

	responseData := make([]interface{}, len(conversations))
	for i, conv := range conversations {
		participants := make([]dto.Participant, len(conv.Participants))
		for j, p := range conv.Participants {
			participants[j] = dto.Participant{
				UserID:      p.UserID.String(),
				UserName:    p.Username,
				UserAvatar:  p.AvatarURL,
				DisplayName: p.DisplayName,
				FirstName:   p.FirstName,
				LastName:    p.LastName,
				AvatarURL:   p.AvatarURL,
				LastSeen: func() string {
					if p.LastSeenAt != nil {
						return p.LastSeenAt.UTC().GoString()
					}
					return ""
				}(),
				OnlineStatus: p.OnlineStatus,
				IsVerified:   p.IsVerified,
			}
		}
		responseData[i] = dto.ConversationResponse{
			ID:               conv.ID.String(),
			Title:            utils.PtrString(conv.Title),
			Description:      utils.PtrString(conv.Description),
			ConversationType: string(conv.ConversationType),
			AvatarURL:        conv.AvatarURL,
			UnreadCount:      conv.UnreadCount,
			MemberCount:      conv.MemberCount,
			MessageCount:     conv.MessageCount,
			LastMessage: func() *dto.Message {
				if conv.LastMessage != nil {
					return &dto.Message{
						Content:      conv.LastMessage.Content,
						MessageType:  string(conv.LastMessage.MessageType),
						SenderUserID: conv.LastMessage.SenderUserID.String(),
						SenderName:   conv.LastMessage.SenderName,
						SenderAvatar: conv.LastMessage.SenderAvatar,
					}
				}
				return nil
			}(),
			Participants: participants,
		}
	}
	response.JSONWithMessage(context, handler.Request(), handler.Writer(), response.StatusOK, "User conversations fetched successfully", responseData)
}
