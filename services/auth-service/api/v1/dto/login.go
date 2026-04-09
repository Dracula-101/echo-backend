package dto

import (
	"auth-service/internal/domain"
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// LoginRequest represents the payload required to authenticate a user.
type LoginRequest struct {
	Email     string  `json:"email" validate:"required,email"`
	Password  string  `json:"password" validate:"required,min=8"`
	FCMToken  *string `json:"fcm_token,omitempty" validate:"omitempty"`
	APNSToken *string `json:"apns_token,omitempty" validate:"omitempty"`
}

// NewLoginRequest constructs a login request with zero values.
func NewLoginRequest() *LoginRequest {
	return &LoginRequest{}
}

// GetValue exposes the request for validation helpers.
func (lr *LoginRequest) GetValue() interface{} {
	return lr
}

// ValidateErrors maps validator errors into transport-friendly details.
func (lr *LoginRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Email":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Invalid email format",
				Code: request.INVALID_FORMAT,
			})
		case "Password":
			if err.Tag() == "required" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Password is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "min" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Password must be at least 8 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "FCMToken", "APNSToken":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Invalid token format",
				Code: request.INVALID_FORMAT,
			})
		}
	}
	return errs, nil
}

// LoginResponse bundles the authenticated user and issued session tokens.
type LoginResponse struct {
	User    LoginUser    `json:"user"`
	Session LoginSession `json:"session"`
}

type AccountStatus string

const (
	AccountStatusActive      AccountStatus = "active"
	AccountStatusPending     AccountStatus = "pending"
	AccountStatusSuspended   AccountStatus = "suspended"
	AccountStatusLocked      AccountStatus = "locked"
	AccountStatusDeactivated AccountStatus = "deactivated"
	AccountStatusDeleted     AccountStatus = "deleted"
)

// LoginUser represents the serialized user details returned to the client.
type LoginUser struct {
	ID               string        `json:"id"`
	Email            string        `json:"email"`
	PhoneNumber      string        `json:"phone_number,omitempty"`
	PhoneCountryCode string        `json:"phone_country_code,omitempty"`
	EmailVerified    bool          `json:"email_verified"`
	PhoneVerified    bool          `json:"phone_verified,omitempty"`
	AccountStatus    AccountStatus `json:"account_status"`
	TFAEnabled       bool          `json:"tfa_enabled"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// LoginSession captures the issued tokens after authentication.
type LoginSession struct {
	SessionId    string    `json:"session_id"`
	SessionToken string    `json:"session_token"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// NewLoginResponse builds a login response from the persisted model state.
func NewLoginResponse(user domain.User, sessionId string, sessionToken string, accessToken string, refreshToken string, expiresAt time.Time) *LoginResponse {
	return &LoginResponse{
		User: LoginUser{
			ID:               user.ID,
			Email:            user.Email,
			PhoneNumber:      user.PhoneNumber,
			PhoneCountryCode: user.PhoneCountryCode,
			EmailVerified:    user.EmailVerified,
			PhoneVerified:    user.PhoneVerified,
			AccountStatus:    AccountStatus(user.AccountStatus),
			TFAEnabled:       user.TwoFactorEnabled,
			CreatedAt:        user.CreatedAt,
			UpdatedAt:        user.UpdatedAt,
		},
		Session: LoginSession{
			SessionId:    sessionId,
			SessionToken: sessionToken,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    string("Bearer"),
			ExpiresAt:    expiresAt,
		},
	}
}
