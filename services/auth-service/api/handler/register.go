package handler

import (
	"auth-service/api/dto"
	"auth-service/internal/service"
	serviceModels "auth-service/internal/service/models"
	"net/http"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	req := dto.NewRegisterRequest()
	if ok := handler.ParseValidateAndSend(req); !ok {
		return
	}
	deviceInfo := request.GetDeviceInfo(r)
	clientIp := request.GetClientIP(r)

	h.log.Info("Register attempt",
		logger.String("email", req.Email),
		logger.String("client_ip", clientIp),
		logger.Any("device_info", deviceInfo),
	)

	emailTaken, dbErr := h.service.IsEmailTaken(r.Context(), req.Email)
	h.log.Debug("Email taken check result",
		logger.String("email", req.Email),
		logger.Bool("email_taken", emailTaken),
	)
	if dbErr != nil {
		h.log.Error("Failed to check if email is taken", logger.Error(dbErr))
		response.InternalServerError(r.Context(), r, w, "Failed to process registration", dbErr)
		return
	}
	if emailTaken {
		response.BadRequestError(r.Context(), r, w, "Email is already registered", nil)
		return
	}

	output, authErr := h.service.RegisterUser(r.Context(), serviceModels.RegisterUserInput{
		Email:            req.Email,
		Password:         req.Password,
		PhoneNumber:      req.PhoneNumber,
		PhoneCountryCode: req.PhoneCountryCode,
		IPAddress:        clientIp,
		UserAgent:        request.GetUserAgent(r),
		AcceptTerms:      req.AcceptTerms,
	})
	if authErr != nil {
		h.log.Error("Failed to register user",
			logger.Any("auth_error", authErr),
		)
		if authErr.Code == service.AuthErrorInvalidCredentials {
			response.BadRequestError(r.Context(), r, w, authErr.Message, nil)
		} else {
			response.InternalServerError(r.Context(), r, w, authErr.Message, authErr.Error)
		}
		return
	}

	h.log.Info("Register successful",
		logger.String("email", req.Email),
		logger.String("client_ip", clientIp),
		logger.Any("output", output),
	)

	response.JSONWithMessage(r.Context(), r, w, http.StatusCreated, "Registration successful",
		dto.NewRegisterResponse(
			output.UserID,
			output.Email,
			output.EmailVerificationSent,
		),
	)
}
