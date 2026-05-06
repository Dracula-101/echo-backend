package protocol

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

// ResumeToken is the opaque value handed to the client on connect and
// presented back on reconnect. It carries the per-conversation seq cursor so
// catch-up is bounded to "everything after this point" instead of the last
// 24 hours.
type ResumeToken struct {
	UserID           string           `json:"u"`
	DeviceID         string           `json:"d"`
	LastDeliveredSeq map[string]int64 `json:"s"`
	IssuedAt         int64            `json:"i"`
	ExpiresAt        int64            `json:"e"`
}

const ResumeTokenTTL = 5 * time.Minute

var (
	ErrResumeTokenExpired = errors.New("resume token expired")
	ErrResumeTokenInvalid = errors.New("resume token invalid")
)

// SignResumeToken returns a base64-encoded "<payload>.<signature>" string.
// The secret is the same JWT secret used by auth-service so we don't have to
// stand up a new one for this purpose.
func SignResumeToken(t *ResumeToken, secret []byte) (string, error) {
	body, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	sig := mac.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(body) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

// VerifyResumeToken decodes and validates the token. A token issued more than
// ResumeTokenTTL ago, or with a bad signature, is rejected.
func VerifyResumeToken(token string, secret []byte) (*ResumeToken, error) {
	dot := -1
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		return nil, ErrResumeTokenInvalid
	}

	bodyB64, sigB64 := token[:dot], token[dot+1:]
	body, err := base64.RawURLEncoding.DecodeString(bodyB64)
	if err != nil {
		return nil, ErrResumeTokenInvalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, ErrResumeTokenInvalid
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return nil, ErrResumeTokenInvalid
	}

	var rt ResumeToken
	if err := json.Unmarshal(body, &rt); err != nil {
		return nil, ErrResumeTokenInvalid
	}
	if time.Now().Unix() > rt.ExpiresAt {
		return nil, ErrResumeTokenExpired
	}
	return &rt, nil
}

// NewResumeToken builds a token. seq may be nil for first-connect tokens
// (no conversations seen yet).
func NewResumeToken(userID, deviceID string, seq map[string]int64) *ResumeToken {
	now := time.Now()
	return &ResumeToken{
		UserID:           userID,
		DeviceID:         deviceID,
		LastDeliveredSeq: seq,
		IssuedAt:         now.Unix(),
		ExpiresAt:        now.Add(ResumeTokenTTL).Unix(),
	}
}
