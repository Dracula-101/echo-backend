package domain

import "time"

// RegistrationInput contains all data needed for user registration
type RegistrationInput struct {
	Email            string
	Password         string
	PhoneNumber      string
	PhoneCountryCode string
	IPAddress        string
	UserAgent        string
	AcceptTerms      bool
}

// LoginInput contains credentials for user authentication
type LoginInput struct {
	Email        string
	Password     string
	IPAddress    string
	UserAgent    string
	DeviceInfo   DeviceInfo
	LocationInfo LocationInfo
}

// SessionCreationInput contains data for creating a new session
type SessionCreationInput struct {
	UserID          string
	RefreshToken    string
	Device          DeviceInfo
	Browser         BrowserInfo
	Location        LocationInfo
	UserAgent       string
	IsTrustedDevice bool
	FCMToken        *string
	APNSToken       *string
	SessionType     string
	ExpiresAt       time.Time
	Metadata        map[string]interface{}
}
