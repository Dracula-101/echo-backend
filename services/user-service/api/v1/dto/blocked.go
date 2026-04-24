package dto

import (
	"time"

	"shared/server/request"
	"user-service/internal/domain"

	"github.com/go-playground/validator/v10"
)

type BlockedUserEntry struct {
	BlockID       string    `json:"block_id"`
	BlockedUserID string    `json:"blocked_user_id"`
	BlockType     string    `json:"block_type"`
	BlockReason   *string   `json:"block_reason,omitempty"`
	BlockedAt     time.Time `json:"blocked_at"`
}

type ListBlockedResponse struct {
	Items []BlockedUserEntry `json:"items"`
	Total int                `json:"total"`
}

func NewListBlockedResponse(items []*domain.BlockedUser) *ListBlockedResponse {
	entries := make([]BlockedUserEntry, 0, len(items))
	for _, b := range items {
		entries = append(entries, BlockedUserEntry{
			BlockID:       b.ID,
			BlockedUserID: b.BlockedUserID,
			BlockType:     string(b.BlockType),
			BlockReason:   b.Reason,
			BlockedAt:     b.BlockedAt,
		})
	}
	return &ListBlockedResponse{
		Items: entries,
		Total: len(entries),
	}
}

type BlockUserRequest struct {
	UserID      string  `json:"user_id" validate:"required,uuid4"`
	BlockReason *string `json:"block_reason,omitempty" validate:"omitempty,max=200"`
	BlockType   string  `json:"block_type,omitempty" validate:"omitempty,oneof=full messages_only"`
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
		case "UserID":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "User ID is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "User ID must be a valid UUIDv4",
					Code: request.INVALID_FORMAT,
				})
			}
		case "BlockReason":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Block reason must be at most 200 characters",
					Code: request.INVALID_FORMAT,
				})
			}
		case "BlockType":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Block type must be 'full' or 'messages_only'",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}
