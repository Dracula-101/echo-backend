package dto

import (
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// RefreshTokenRequest represents the payload required to refresh authentication tokens.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// NewRefreshTokenRequest constructs a refresh token request with zero values.
func NewRefreshTokenRequest() *RefreshTokenRequest {
	return &RefreshTokenRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *RefreshTokenRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *RefreshTokenRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "RefreshToken":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Refresh token is required",
				Code: request.REQUIRED_FIELD,
			})
		}
	}
	return errs, nil
}

// RefreshTokenResponse bundles the new access token and optionally a new refresh token.
type RefreshTokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// NewRefreshTokenResponse builds a refresh token response.
func NewRefreshTokenResponse(accessToken string, refreshToken string, expiresAt time.Time) *RefreshTokenResponse {
	return &RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
	}
}
