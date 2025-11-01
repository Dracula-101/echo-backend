package cache

import (
	"time"
)

const (
	NoExpiration      time.Duration = -1
	DefaultExpiration time.Duration = 0
)

const (
	TTL1Minute   = 1 * time.Minute
	TTL5Minutes  = 5 * time.Minute
	TTL15Minutes = 15 * time.Minute
	TTL30Minutes = 30 * time.Minute
	TTL1Hour     = 1 * time.Hour
	TTL24Hours   = 24 * time.Hour
	TTL7Days     = 7 * 24 * time.Hour
)
