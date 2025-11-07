package token

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

type Claims struct {
	jwt.RegisteredClaims
	TokenType TokenType      `json:"token_type"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Raw       map[string]any `json:"-"`
	IssuedKey string         `json:"-"`
}

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

func (c *Claims) Capture(raw jwt.Claims) {
	if raw == nil {
		return
	}
	bytes, err := json.Marshal(raw)
	if err != nil {
		return
	}
	_ = json.Unmarshal(bytes, &c.Raw)
}

func (c *Claims) Validate(now time.Time, leeway time.Duration, expect TokenType, expectedAudience []string) error {
	if expect != "" && c.TokenType != expect {
		return errors.New("token: unexpected token type")
	}
	if c.ExpiresAt != nil && now.After(c.ExpiresAt.Time.Add(leeway)) {
		return ErrExpiredToken
	}
	if c.NotBefore != nil && now.Add(leeway).Before(c.NotBefore.Time) {
		return errors.New("token: token not yet valid")
	}
	if len(expectedAudience) > 0 && !hasAudience(c.Audience, expectedAudience) {
		return errors.New("token: audience mismatch")
	}
	return nil
}

func hasAudience(actual jwt.ClaimStrings, expected []string) bool {
	if len(actual) == 0 {
		return false
	}
	for _, want := range expected {
		for _, got := range actual {
			if got == want {
				return true
			}
		}
	}
	return false
}
