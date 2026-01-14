package domain

// LoginResult contains the result of a successful login
type LoginResult struct {
	User         *User
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	Session      *Session
}

// PasswordHashResult contains the result of password hashing
type PasswordHashResult struct {
	Hash      string
	Salt      []byte
	Algorithm string
}
