package dto

import (
	"echo-backend/services/message-service/internal/models"
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type GetMessagesRequest struct {
	ConversationID string `json:"conversation_id" validate:"required,uuid4"`
	Limit          int    `json:"limit" validate:"omitempty,min=1,max=100"`
	Offset         int    `json:"offset" validate:"omitempty,min=0"`
}

func NewGetMessagesRequest() *GetMessagesRequest {
	return &GetMessagesRequest{
		Limit:  20,
		Offset: 0,
	}
}

func (r *GetMessagesRequest) GetValue() interface{} {
	return r
}

func (r *GetMessagesRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "ConversationID":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "conversation_id_required",
					Msg:  "Conversation ID is required",
				})
			} else if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "invalid_conversation_id",
					Msg:  "Conversation ID must be a valid UUIDv4",
				})
			}
		case "Limit":
			if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "invalid_limit",
					Msg:  "Limit must be at least 1",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "invalid_limit",
					Msg:  "Limit must be at most 100",
				})
			}
		case "Offset":
			if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "invalid_offset",
					Msg:  "Offset must be at least 0",
				})
			}
		default:
			errors = append(errors, request.ValidationErrorDetail{
				Code: fieldErr.Field(),
				Msg:  "Invalid value",
			})
		}
	}
	return errors, nil
}

type GetMessagesResponse struct {
	models.MessagesResponse
}
