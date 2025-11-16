package dto

import (
	"presence-service/internal/model"
	"shared/server/request"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type BroadcastEventRequest struct {
	EventType  model.EventType `json:"event_type" validate:"required"`
	Recipients []string        `json:"recipients" validate:"required,min=1,dive,uuid"`
	Sender     *string         `json:"sender,omitempty" validate:"omitempty,uuid"`
	Payload    interface{}     `json:"payload" validate:"required"`
	Priority   int             `json:"priority,omitempty" validate:"omitempty,min=0,max=2"`
	TTL        int             `json:"ttl,omitempty" validate:"omitempty,min=0"`
}

func NewBroadcastEventRequest() *BroadcastEventRequest {
	return &BroadcastEventRequest{}
}

func (r *BroadcastEventRequest) GetValue() interface{} {
	return r
}

func (r *BroadcastEventRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		field := err.Field()
		switch field {
		case "EventType":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Event type is required",
					Code: request.REQUIRED_FIELD,
				})
			}
		case "Recipients":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Recipients are required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "min" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "At least one recipient is required",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Payload":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Payload is required",
					Code: request.REQUIRED_FIELD,
				})
			}
		}
	}
	return msgs, nil
}

// ToModel converts DTO to model with UUID parsing
func (r *BroadcastEventRequest) ToModel() (*model.BroadcastRequest, error) {
	recipients := make([]uuid.UUID, len(r.Recipients))
	for i, recipientStr := range r.Recipients {
		recipientID, err := uuid.Parse(recipientStr)
		if err != nil {
			return nil, err
		}
		recipients[i] = recipientID
	}

	var sender *uuid.UUID
	if r.Sender != nil {
		senderID, err := uuid.Parse(*r.Sender)
		if err != nil {
			return nil, err
		}
		sender = &senderID
	}

	return &model.BroadcastRequest{
		EventType:  r.EventType,
		Recipients: recipients,
		Sender:     sender,
		Payload:    r.Payload,
		Priority:   r.Priority,
		TTL:        r.TTL,
	}, nil
}
