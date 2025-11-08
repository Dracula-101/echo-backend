package handler

import (
	"auth-service/api/dto"
	authErrors "auth-service/internal/errors"
	serviceModels "auth-service/internal/service/models"
	"net/http"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Registration request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	req := dto.NewRegisterRequest()
	if ok := handler.ParseValidateAndSend(req); !ok {
		h.log.Warn("Registration request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	clientIp := handler.GetClientIP()

	h.log.Debug("Checking if email is already registered",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("email", req.Email),
	)

	emailTaken, dbErr := h.service.IsEmailTaken(r.Context(), req.Email)
	if dbErr != nil {
		h.log.Error("Failed to check if email is taken",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", req.Email),
			logger.Error(dbErr),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to process registration", dbErr)
		return
	}
	if emailTaken {
		h.log.Warn("Registration attempt with existing email",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", req.Email),
		)
		response.BadRequestError(r.Context(), r, w, "Email is already registered", nil)
		return
	}

	output, authErr := h.service.RegisterUser(r.Context(), serviceModels.RegisterUserInput{
		Email:            req.Email,
		Password:         req.Password,
		PhoneNumber:      req.PhoneNumber,
		PhoneCountryCode: req.PhoneCountryCode,
		IPAddress:        clientIp,
		UserAgent:        handler.GetUserAgent(),
		AcceptTerms:      req.AcceptTerms,
	})
	if authErr != nil {
		h.log.Error("Failed to register user",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", req.Email),
			logger.String("error_code", authErr.Code),
			logger.Error(authErr.Error),
		)
		if authErr.Code == authErrors.CodePasswordTooWeak || authErr.Code == authErrors.CodeInvalidEmail {
			response.BadRequestError(r.Context(), r, w, authErr.Message, nil)
		} else {
			response.InternalServerError(r.Context(), r, w, authErr.Message, authErr.Error)
		}
		return
	}

	h.log.Info("User registered successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("user_id", output.UserID),
		logger.String("email", output.Email),
	)

	response.JSONWithMessage(r.Context(), r, w, http.StatusCreated, "Registration successful",
		dto.NewRegisterResponse(
			output.UserID,
			output.Email,
			output.EmailVerificationSent,
		),
	)
}
