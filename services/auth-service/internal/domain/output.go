package domain

// LoginResult contains the result of a successful login
type LoginResult struct {
	User         *User
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

type RegisterUserOutput struct {
	UserID                string
	Email                 string
	EmailVerificationSent bool
	VerificationToken     string
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
