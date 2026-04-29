package dto

import (
	"auth-service/internal/domain"
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

type LoginRequest struct {
	Email       *string `json:"email,omitempty"        validate:"omitempty,email"`
	Password    *string `json:"password,omitempty"     validate:"omitempty,min=8"`
	VerifyToken *string `json:"verify_token,omitempty"`
	FCMToken    *string `json:"fcm_token,omitempty"`
	APNSToken   *string `json:"apns_token,omitempty"`
}

func NewLoginRequest() *LoginRequest {
	return &LoginRequest{}
}

func (lr *LoginRequest) GetValue() interface{} {
	return lr
}

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
			if err.Tag() == "min" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Password must be at least 8 characters",
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

func (lr *LoginRequest) Validate() ([]request.ValidationErrorDetail, error) {
	if lr.VerifyToken != nil {
		return nil, nil
	}

	if lr.Email == nil {
		return []request.ValidationErrorDetail{{
			Msg:  "Email is required",
			Code: request.REQUIRED_FIELD,
		}}, nil
	}

	if lr.Password == nil {
		return []request.ValidationErrorDetail{{
			Msg:  "Password is required",
			Code: request.REQUIRED_FIELD,
		}}, nil
	}

	return nil, nil
}

// LoginResponse bundles the authenticated user and issued session tokens.
type LoginResponse struct {
	User     LoginUser     `json:"user"`
	Session  LoginSession  `json:"session"`
	Location LoginLocation `json:"location,omitempty"`
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
	SessionId       string    `json:"session_id"`
	SessionToken    string    `json:"session_token"`
	AccessToken     string    `json:"access_token"`
	RefreshToken    string    `json:"refresh_token,omitempty"`
	TokenType       string    `json:"token_type"`
	ExpiresAt       time.Time `json:"expires_at"`
	ExpiresIn       int64     `json:"expires_in"`
	IsNewLocation   bool      `json:"is_new_location"`
	IsTrustedDevice bool      `json:"is_trusted_device"`
}

type LoginLocation struct {
	City    string `json:"city,omitempty"`
	Region  string `json:"region,omitempty"`
	Country string `json:"country,omitempty"`
}

// NewLoginResponse builds a login response from the persisted model state.
func NewLoginResponse(user domain.User, sessionId string, sessionToken string, accessToken string, refreshToken string, isNewLocation bool, isTrustedDevice bool, expiresAt time.Time, expiresIn int64, location request.IpAddressInfo) *LoginResponse {
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
			SessionId:       sessionId,
			SessionToken:    sessionToken,
			AccessToken:     accessToken,
			RefreshToken:    refreshToken,
			TokenType:       string("Bearer"),
			ExpiresAt:       expiresAt,
			IsNewLocation:   isNewLocation,
			IsTrustedDevice: isTrustedDevice,
			ExpiresIn:       expiresIn,
		},
		Location: LoginLocation{
			City:    location.City,
			Region:  location.State,
			Country: location.Country,
		},
	}
}
