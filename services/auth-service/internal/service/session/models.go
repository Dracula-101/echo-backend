package session

import (
	"encoding/json"
	"time"
)

type SessionData struct {
	UserID        string
	SessionID     string
	SessionToken  string
	DeviceID      string
	ExpiresAt     int64
	AccountStatus string
}

func (s *SessionData) Bytes() []byte {
	data, _ := json.Marshal(s)
	return data
}

func (s *SessionData) FromBytes(data []byte) error {
	return json.Unmarshal(data, s)
}

func (s *SessionData) IsExpired() bool {
	return time.Unix(s.ExpiresAt, 0).Before(time.Now())
}

type Pre2FAData struct {
	UserID    string
	DeviceID  string
	IpAddress string
	CreatedAt int64
}

func (p *Pre2FAData) Bytes() []byte {
	data, _ := json.Marshal(p)
	return data
}

func (p *Pre2FAData) FromBytes(data []byte) error {
	return json.Unmarshal(data, p)
}

func (p *Pre2FAData) IsExpired() bool {
	return time.Unix(p.CreatedAt, 0).Add(PreAuthTTL).Before(time.Now())
}

type Setup2FAData struct {
	UserID       string
	SecretBase32 string
	BackupCodes  []string
}

func (s *Setup2FAData) Bytes() []byte {
	data, _ := json.Marshal(s)
	return data
}

func (s *Setup2FAData) FromBytes(data []byte) error {
	return json.Unmarshal(data, s)
}

type UserStatusData struct {
	AccountStatus    string
	TwoFactorEnabled bool
}

func (u *UserStatusData) Bytes() []byte {
	data, _ := json.Marshal(u)
	return data
}

func (u *UserStatusData) FromBytes(data []byte) error {
	return json.Unmarshal(data, u)
}
