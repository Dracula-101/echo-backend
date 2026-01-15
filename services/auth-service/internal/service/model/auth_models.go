package model

import "shared/server/request"

type RegisterUserInput struct {
	Email            string
	Password         string
	PhoneNumber      string
	PhoneCountryCode string
	IPAddress        string
	UserAgent        string
	AcceptTerms      bool
}

type RegisterUserOutput struct {
	UserID                string
	Email                 string
	EmailVerificationSent bool
	VerificationToken     string
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginUser struct {
	ID               string
	Email            string
	PhoneNumber      string
	PhoneCountryCode string
	EmailVerified    bool
	PhoneVerified    bool
	AccountStatus    string
	TFAEnabled       bool
	CreatedAt        int64
	UpdatedAt        int64
}

type LoginSession struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    int64
}

type LoginResult struct {
	User    LoginUser
	Session LoginSession
}

type FailedLoginAttemptInput struct {
	UserID        string
	Device        request.DeviceInfo
	Location      *request.IpAddressInfo
	UserAgent     string
	Reason        string
	LoginMethod   string
	IsNewDevice   bool
	IsNewLocation bool
}
