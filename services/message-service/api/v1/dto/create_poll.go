package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type CreatePollRequest struct {
	ConversationID       string   `json:"conversation_id" validate:"required,uuid4"`
	Question             string   `json:"question" validate:"required,min=1,max=255"`
	Options              []string `json:"options" validate:"required,min=2,max=10,dive,min=1,max=100"`
	AllowMultipleAnswers bool     `json:"allow_multiple_answers"`
	IsAnonymous          bool     `json:"is_anonymous"`
	IsQuiz               bool     `json:"is_quiz"`
	CorrectOptionIndex   *int     `json:"correct_option_index,omitempty" validate:"omitempty,gte=0"`
	Explanation          *string  `json:"explanation,omitempty" validate:"omitempty,max=200"`
	ClosesAt             *string  `json:"closes_at,omitempty"`
}

func NewCreatePollRequest() *CreatePollRequest {
	return &CreatePollRequest{}
}

func (r *CreatePollRequest) GetValue() interface{} {
	return r
}

func (r *CreatePollRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	errors := make([]request.ValidationErrorDetail, 0, len(ve))
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "ConversationID":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.REQUIRED_FIELD, Msg: "conversation_id is required"})
			} else if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.INVALID_FORMAT, Msg: "conversation_id must be a valid UUID"})
			}
		case "Question":
			switch fieldErr.Tag() {
			case "required", "min":
				errors = append(errors, request.ValidationErrorDetail{Code: request.REQUIRED_FIELD, Msg: "question is required"})
			case "max":
				errors = append(errors, request.ValidationErrorDetail{Code: request.TOO_LONG, Msg: "question must be at most 255 characters"})
			}
		case "Options":
			switch fieldErr.Tag() {
			case "required", "min":
				errors = append(errors, request.ValidationErrorDetail{Code: request.INVALID_FORMAT, Msg: "options must contain at least 2 entries"})
			case "max":
				errors = append(errors, request.ValidationErrorDetail{Code: request.TOO_LONG, Msg: "options must contain at most 10 entries"})
			}
		case "Explanation":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.TOO_LONG, Msg: "explanation must be at most 200 characters"})
			}
		}
	}
	return errors, nil
}
