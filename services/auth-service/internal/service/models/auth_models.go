package models

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
