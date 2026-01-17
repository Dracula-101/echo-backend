package utils

import "time"

func IsValidTime(t time.Time) bool {
	now := time.Now()
	return !t.IsZero() && t.Before(now)
}
