package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// BlockUserRequest represents a request to block a user
type BlockUserRequest struct {
	BlockedUserID string  `json:"blocked_user_id" validate:"required,uuid4"`
	Reason        *string `json:"reason,omitempty" validate:"omitempty,max=255"`
}

func NewBlockUserRequest() *BlockUserRequest {
	return &BlockUserRequest{}
}

func (r *BlockUserRequest) GetValue() interface{} {
	return r
}

func (r *BlockUserRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "BlockedUserID":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Blocked user ID is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Blocked user ID must be a valid UUIDv4",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Reason":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Reason must be at most 255 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}
