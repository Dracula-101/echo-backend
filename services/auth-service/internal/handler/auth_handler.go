package handler

import (
	"auth-service/internal/service"
	"net/http"
	"shared/pkg/logger"
	"shared/server/response"
)

type AuthHandler struct {
	service *service.AuthService
	log     logger.Logger
}

func NewAuthHandler(service *service.AuthService, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		service: service,
		log:     log,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Register endpoint", nil)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Login endpoint", nil)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Logout endpoint", nil)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Refresh token endpoint", nil)
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Verify email endpoint", nil)
}

func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Resend verification endpoint", nil)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Forgot password endpoint", nil)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Reset password endpoint", nil)
}
