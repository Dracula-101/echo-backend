package utils

import (
	"errors"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(
	`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+$`,
)

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

type EmailInfo struct {
	Email     string
	Domain    string
	LocalPart string
}

func ValidateEmail(email string) (*EmailInfo, error) {
	if len(email) == 0 {
		return nil, errors.New("email is required")
	}

	if len(email) > 255 {
		return nil, errors.New("email must not exceed 255 characters")
	}

	if !emailRegex.MatchString(email) {
		return nil, errors.New("invalid email format")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return nil, errors.New("invalid email format")
	}

	local := parts[0]
	domain := parts[1]

	if strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") {
		return nil, errors.New("local part cannot start or end with dot")
	}

	if strings.Contains(local, "..") {
		return nil, errors.New("local part cannot contain consecutive dots")
	}

	if !strings.Contains(domain, ".") {
		return nil, errors.New("domain must contain at least one dot")
	}

	return &EmailInfo{
		Email:     email,
		Domain:    domain,
		LocalPart: local,
	}, nil
}

func MaskEmail(email string) string {
	email = NormalizeEmail(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	local := parts[0]
	domain := parts[1]

	maskedLocal := local
	if len(local) > 2 {
		maskedLocal = local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:]
	} else if len(local) == 2 {
		maskedLocal = local[:1] + "*"
	}

	return maskedLocal + "@" + domain
}
