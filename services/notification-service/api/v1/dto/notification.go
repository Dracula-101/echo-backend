package dto

import (
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// List Notifications
// ============================================================================

// ListNotificationsRequest represents query parameters for listing notifications.
type ListNotificationsRequest struct {
	Cursor string `json:"cursor" validate:"omitempty"`
	Limit  int    `json:"limit" validate:"omitempty,min=1,max=100"`
	Type   string `json:"type" validate:"omitempty"`
	Unread *bool  `json:"unread" validate:"omitempty"`
}

func NewListNotificationsRequest() *ListNotificationsRequest {
	return &ListNotificationsRequest{}
}

func (r *ListNotificationsRequest) GetValue() interface{} {
	return r
}

func (r *ListNotificationsRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Limit":
			msgs = append(msgs, request.ValidationErrorDetail{
				Msg:  "Limit must be between 1 and 100",
				Code: request.INVALID_FORMAT,
			})
		}
	}
	return msgs, nil
}

// NotificationResponse represents a single notification in responses.
type NotificationResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Data      any        `json:"data,omitempty"`
	Read      bool       `json:"read"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ListNotificationsResponse represents the paginated list of notifications.
type ListNotificationsResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	NextCursor    string                 `json:"next_cursor,omitempty"`
	HasMore       bool                   `json:"has_more"`
}

// ============================================================================
// Mark As Read
// ============================================================================

// MarkAsReadRequest represents the payload to mark specific notifications as read.
type MarkAsReadRequest struct {
	NotificationIDs []string `json:"notification_ids" validate:"required,min=1,dive,required"`
}

func NewMarkAsReadRequest() *MarkAsReadRequest {
	return &MarkAsReadRequest{}
}

func (r *MarkAsReadRequest) GetValue() interface{} {
	return r
}

func (r *MarkAsReadRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "NotificationIDs":
			if err.Tag() == "required" || err.Tag() == "min" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "At least one notification ID is required",
					Code: request.REQUIRED_FIELD,
				})
			}
		}
	}
	return msgs, nil
}
