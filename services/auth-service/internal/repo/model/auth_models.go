package model

// CreateUserParams represents the data required to create a new user record.
type CreateUserParams struct {
	Email             string
	PasswordHash      string
	PasswordSalt      string
	PasswordAlgorithm string
	PhoneNumber       string
	PhoneCountryCode  string
	IPAddress         string
	UserAgent         string
}
