package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// Send Notification
// ============================================================================

// SendNotificationRequest represents the payload for sending a single notification.
type SendNotificationRequest struct {
	UserID   string `json:"user_id" validate:"required"`
	Type     string `json:"type" validate:"required"`
	Title    string `json:"title" validate:"required,max=255"`
	Body     string `json:"body" validate:"required,max=4096"`
	Data     any    `json:"data,omitempty"`
	Priority string `json:"priority" validate:"omitempty,oneof=low normal high urgent"`
	Channel  string `json:"channel" validate:"omitempty,oneof=push email sms in_app"`
}

func NewSendNotificationRequest() *SendNotificationRequest {
	return &SendNotificationRequest{}
}

func (r *SendNotificationRequest) GetValue() interface{} {
	return r
}

func (r *SendNotificationRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "UserID":
			msgs = append(msgs, request.ValidationErrorDetail{
				Msg:  "User ID is required",
				Code: request.REQUIRED_FIELD,
			})
		case "Type":
			msgs = append(msgs, request.ValidationErrorDetail{
				Msg:  "Notification type is required",
				Code: request.REQUIRED_FIELD,
			})
		case "Title":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Title is required",
					Code: request.REQUIRED_FIELD,
				})
			} else {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Title must be at most 255 characters",
					Code: request.TOO_LONG,
				})
			}
		case "Body":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Body is required",
					Code: request.REQUIRED_FIELD,
				})
			} else {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Body must be at most 4096 characters",
					Code: request.TOO_LONG,
				})
			}
		case "Priority":
			msgs = append(msgs, request.ValidationErrorDetail{
				Msg:  "Priority must be one of: low, normal, high, urgent",
				Code: request.INVALID_FORMAT,
			})
		case "Channel":
			msgs = append(msgs, request.ValidationErrorDetail{
				Msg:  "Channel must be one of: push, email, sms, in_app",
				Code: request.INVALID_FORMAT,
			})
		}
	}
	return msgs, nil
}

// SendNotificationResponse represents the response after sending a notification.
type SendNotificationResponse struct {
	NotificationID string `json:"notification_id"`
	Status         string `json:"status"`
}

// ============================================================================
// Send Bulk Notifications
// ============================================================================

// SendBulkNotificationsRequest represents the payload for sending bulk notifications.
type SendBulkNotificationsRequest struct {
	Notifications []SendNotificationRequest `json:"notifications" validate:"required,min=1,dive"`
}

func NewSendBulkNotificationsRequest() *SendBulkNotificationsRequest {
	return &SendBulkNotificationsRequest{}
}

func (r *SendBulkNotificationsRequest) GetValue() interface{} {
	return r
}

func (r *SendBulkNotificationsRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Notifications":
			if err.Tag() == "required" || err.Tag() == "min" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "At least one notification is required",
					Code: request.REQUIRED_FIELD,
				})
			}
		}
	}
	return msgs, nil
}

// SendBulkNotificationsResponse represents the response after sending bulk notifications.
type SendBulkNotificationsResponse struct {
	Sent   int      `json:"sent"`
	Failed int      `json:"failed"`
	Errors []string `json:"errors,omitempty"`
}
