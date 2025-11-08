package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"shared/server/env"
	"shared/server/headers"
	"shared/server/response"

	"github.com/go-playground/validator/v10"
)

type RequestHandler struct {
	config    *Config
	validator *validator.Validate
	request   *http.Request
	writer    http.ResponseWriter
}

func NewHandler(req *http.Request, writer http.ResponseWriter) *RequestHandler {
	return &RequestHandler{
		config: &Config{
			MaxBodySize:        DefaultMaxBodySize,
			DisallowUnknown:    true,
			RequireContentType: true,
			AllowEmptyBody:     false,
		},
		validator: validator.New(),
		request:   req,
		writer:    writer,
	}
}

// Configuration methods

func (h *RequestHandler) AllowEmptyBody() *RequestHandler {
	h.config.AllowEmptyBody = true
	return h
}

func (h *RequestHandler) WithConfig(config *Config) *RequestHandler {
	h.config = config
	return h
}

func (h *RequestHandler) WithMaxBodySize(size int64) *RequestHandler {
	h.config.MaxBodySize = size
	return h
}

func (h *RequestHandler) WithAllowUnknown() *RequestHandler {
	h.config.DisallowUnknown = false
	return h
}

func (h *RequestHandler) RegisterValidation(tag string, fn validator.Func) error {
	return h.validator.RegisterValidation(tag, fn)
}

// Validation methods

type FieldErrorDetail struct {
	Field   string
	Tag     string
	Value   string
	Message string
}

var errEmptyBody = errors.New("request body is empty")

func (h *RequestHandler) ParseAndValidate(req Validator) ([]response.FieldError, error) {
	if err := h.validateRequest(); err != nil {
		return nil, err
	}

	if err := h.parseJSON(req.GetValue()); err != nil {
		if errors.Is(err, errEmptyBody) && h.config.AllowEmptyBody {
			return nil, nil
		}
		return nil, err
	}

	if err := h.validator.Struct(req.GetValue()); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			msgs, customErr := req.ValidateErrors(validationErrors)
			if customErr != nil {
				return nil, customErr
			}
			var fieldErrors []response.FieldError
			for index, err := range validationErrors {
				var constraints string
				if env.IsDevelopment() {
					constraints = fmt.Sprintf("%s has a constraint of %s in %s", err.Field(), err.Tag(), err.Namespace())
				} else {
					constraints = ""
				}
				fieldErrors = append(fieldErrors, response.FieldError{
					Field:       err.Field(),
					Value:       fmt.Sprintf("%v", err.Value()),
					Message:     msgs[index].Msg,
					Code:        msgs[index].Code,
					Constraints: constraints,
				})
			}
			return fieldErrors, nil
		}
		return nil, err
	}

	return nil, nil
}

func (h *RequestHandler) ParseValidateAndSend(req Validator) bool {
	validationErr, err := h.ParseAndValidate(req)
	if err != nil && len(validationErr) == 0 {
		response.Error().
			WithRequest(h.request).
			WithMessage("Invalid request").
			WithError(&response.ErrorDetails{
				Code:        "INVALID_REQUEST",
				Type:        "Bad Request",
				InnerError:  err.Error(),
				Message:     "Failed to parse and validate request",
				Description: "Ensure the request body is valid JSON and meets all validation criteria",
			}).
			BadRequest(h.writer)
		return false
	} else {
		if len(validationErr) > 0 {
			response.Error().
				WithRequest(h.request).
				WithMessage("Validation failed").
				WithError(&response.ErrorDetails{
					Code:        "VALIDATION_FAILED",
					Type:        "ValidationError",
					InnerError:  "One or more fields failed validation",
					Message:     "Request validation errors",
					Description: "Check the validation messages for details",
					Fields:      validationErr,
				}).
				BadRequest(h.writer)
			return false
		}
	}
	return true
}

func (h *RequestHandler) validateRequest() error {
	if h.config.RequireContentType {
		contentType := h.request.Header.Get(headers.ContentType)
		if contentType != "" && contentType != headers.ApplicationJSON {
			return fmt.Errorf("%s", fmt.Sprintf("content-Type must be %s", headers.ApplicationJSON))
		}
	}

	if h.request.Body == nil && !h.config.AllowEmptyBody {
		return fmt.Errorf("request body is required")
	}

	return nil
}

func (h *RequestHandler) parseJSON(v interface{}) error {
	if h.request.Body == nil {
		return errEmptyBody
	}

	h.request.Body = http.MaxBytesReader(nil, h.request.Body, h.config.MaxBodySize)

	decoder := json.NewDecoder(h.request.Body)
	if h.config.DisallowUnknown {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(v); err != nil {
		if err == io.EOF {
			return errEmptyBody
		}
		if jsonErr, ok := err.(*json.SyntaxError); ok {
			return fmt.Errorf("invalid JSON at position %d", jsonErr.Offset)
		}
		if jsonErr, ok := err.(*json.UnmarshalTypeError); ok {
			return fmt.Errorf("invalid type for field %s: expected %s", jsonErr.Field, jsonErr.Type)
		}
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	if decoder.More() {
		return fmt.Errorf("request body must contain only one JSON object")
	}

	return nil
}
