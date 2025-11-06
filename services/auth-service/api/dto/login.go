package dto

import (
	"auth-service/internal/model"
	dbModel "shared/pkg/database/postgres/models"
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type LoginRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FCMToken  string `json:"fcm_token,omitempty" validate:"omitempty"`
	APNSToken string `json:"apns_token,omitempty" validate:"omitempty"`
}

func NewLoginRequest() *LoginRequest {
	return &LoginRequest{}
}

func (lr *LoginRequest) GetValue() interface{} {
	return lr
}

func (lr *LoginRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Email":
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "Invalid email format",
				Code: request.INVALID_FORMAT,
			})
		case "Password":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Password is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Password must be at least 8 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}

type LoginResponse struct {
	User    User    `json:"user"`
	Session Session `json:"session"`
}

type User struct {
	ID               string                `json:"id"`
	Email            string                `json:"email"`
	PhoneNumber      string                `json:"phone_number,omitempty"`
	PhoneCountryCode string                `json:"phone_country_code,omitempty"`
	EmailVerified    bool                  `json:"email_verified"`
	PhoneVerified    bool                  `json:"phone_verified,omitempty"`
	AccountStatus    dbModel.AccountStatus `json:"account_status"`
	TFAEnabled       bool                  `json:"tfa_enabled"`
	CreatedAt        int64                 `json:"created_at"`
	UpdatedAt        int64                 `json:"updated_at"`
}

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresAt    int64  `json:"expires_at"`
}

func NewLoginResponse(user model.User, accessToken string, refreshToken string, expiresAt int64) *LoginResponse {
	return &LoginResponse{
		User: User{
			ID:               user.ID,
			Email:            user.Email,
			PhoneNumber:      *user.PhoneNumber,
			PhoneCountryCode: *user.PhoneCountryCode,
			EmailVerified:    user.EmailVerified,
			PhoneVerified:    user.PhoneVerified,
			AccountStatus:    user.AccountStatus,
			TFAEnabled:       user.TwoFactorEnabled,
			CreatedAt:        user.CreatedAt.Unix(),
			UpdatedAt:        user.UpdatedAt.Unix(),
		},
		Session: Session{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresAt:    expiresAt,
		},
	}
}
