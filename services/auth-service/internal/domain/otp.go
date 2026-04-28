package domain

import "time"

type OTPVerification struct {
	Phone     string
	OTPHash   string
	Attempts  int
	ExpiresAt time.Time
	SentVia   string // e.g., "sms"
}

func (o *OTPVerification) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}
