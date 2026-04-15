package domain

import "time"

// LoginResult contains the result of a successful login
type LoginResult struct {
	User         *User
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	ExpiresIn    int64
}

type RegisterUserOutput struct {
	UserID                  string
	Email                   string
	EmailVerified           bool
	PhoneVerified           bool
	AccountStatus           string
	NextStep                string
	VerificationEmailSentTo *string
}

type RefreshTokenResult struct {
	User
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    time.Time
}

// PasswordHashResult contains the result of password hashing
type PasswordHashResult struct {
	Hash      string
	Salt      []byte
	Algorithm string
}

type CreateSessionOutput struct {
	SessionId    string
	SessionToken string
}

type MFAEnableOutput struct {
	Secret    string
	QRCodeURL string
}
