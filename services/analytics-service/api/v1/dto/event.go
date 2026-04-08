package dto

import (
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// Track Event
// ============================================================================

// TrackEventRequest represents the payload for tracking a single event.
type TrackEventRequest struct {
	EventType  string            `json:"event_type" validate:"required,max=255"`
	UserID     string            `json:"user_id" validate:"required"`
	SessionID  string            `json:"session_id" validate:"omitempty"`
	Properties map[string]any    `json:"properties,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timestamp  *time.Time        `json:"timestamp,omitempty"`
}

func NewTrackEventRequest() *TrackEventRequest {
	return &TrackEventRequest{}
}

func (r *TrackEventRequest) GetValue() interface{} {
	return r
}

func (r *TrackEventRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "EventType":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Event type is required",
					Code: request.REQUIRED_FIELD,
				})
			} else {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Event type must be at most 255 characters",
					Code: request.TOO_LONG,
				})
			}
		case "UserID":
			msgs = append(msgs, request.ValidationErrorDetail{
				Msg:  "User ID is required",
				Code: request.REQUIRED_FIELD,
			})
		}
	}
	return msgs, nil
}

// TrackEventResponse represents the response after tracking an event.
type TrackEventResponse struct {
	EventID   string    `json:"event_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ============================================================================
// Track Batch Events
// ============================================================================

// TrackBatchEventsRequest represents the payload for tracking multiple events.
type TrackBatchEventsRequest struct {
	Events []TrackEventRequest `json:"events" validate:"required,min=1,max=1000,dive"`
}

func NewTrackBatchEventsRequest() *TrackBatchEventsRequest {
	return &TrackBatchEventsRequest{}
}

func (r *TrackBatchEventsRequest) GetValue() interface{} {
	return r
}

func (r *TrackBatchEventsRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Events":
			if err.Tag() == "required" || err.Tag() == "min" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "At least one event is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "max" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Cannot exceed 1000 events per batch",
					Code: request.TOO_LONG,
				})
			}
		}
	}
	return msgs, nil
}

// TrackBatchEventsResponse represents the response after tracking batch events.
type TrackBatchEventsResponse struct {
	Accepted int      `json:"accepted"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

// ============================================================================
// Query Events
// ============================================================================

// QueryEventsRequest represents query parameters for querying events.
type QueryEventsRequest struct {
	UserID    string `json:"user_id" validate:"omitempty"`
	EventType string `json:"event_type" validate:"omitempty"`
	StartDate string `json:"start_date" validate:"omitempty"`
	EndDate   string `json:"end_date" validate:"omitempty"`
	Cursor    string `json:"cursor" validate:"omitempty"`
	Limit     int    `json:"limit" validate:"omitempty,min=1,max=1000"`
}

func NewQueryEventsRequest() *QueryEventsRequest {
	return &QueryEventsRequest{}
}

func (r *QueryEventsRequest) GetValue() interface{} {
	return r
}

func (r *QueryEventsRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Limit":
			msgs = append(msgs, request.ValidationErrorDetail{
				Msg:  "Limit must be between 1 and 1000",
				Code: request.INVALID_FORMAT,
			})
		}
	}
	return msgs, nil
}

// EventResponse represents a single event in query responses.
type EventResponse struct {
	ID         string            `json:"id"`
	EventType  string            `json:"event_type"`
	UserID     string            `json:"user_id"`
	SessionID  string            `json:"session_id,omitempty"`
	Properties map[string]any    `json:"properties,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	CreatedAt  time.Time         `json:"created_at"`
}

// QueryEventsResponse represents the paginated list of events.
type QueryEventsResponse struct {
	Events     []EventResponse `json:"events"`
	NextCursor string          `json:"next_cursor,omitempty"`
	HasMore    bool            `json:"has_more"`
	Total      int64           `json:"total"`
}
