package domain

import "time"

// LoginHistory represents a login attempt record
type LoginHistory struct {
	ID            string
	UserID        string
	SessionID     *string
	LoginMethod   string
	Status        string
	IPAddress     string
	City          string
	Region        string
	Country       string
	DeviceType    string
	DeviceOS      string
	BrowserName   string
	UserAgent     string
	FailureReason *string
	IsNewDevice   bool
	IsNewLocation bool
	Timestamp     time.Time
}

func (lh *LoginHistory) IsSuccessful() bool {
	return lh.Status == "success"
}

func (lh *LoginHistory) IsFailure() bool {
	return lh.Status == "failure"
}

func (lh *LoginHistory) TimeSinceLogin() time.Duration {
	return time.Since(lh.Timestamp)
}
