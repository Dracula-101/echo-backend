package ratelimiter

import "time"

const (
	RLProfileUpdatePrefix = "rl:profile_update:"
	RLContactAddPrefix    = "rl:contact_add:"
	RLBlockPrefix         = "rl:block:"
	RLSearchPrefix        = "rl:search:"
	RLReportPrefix        = "rl:report:"
	RLStatusCreatePrefix  = "rl:status_create:"
	RLStatusViewPrefix    = "rl:status_view:"
	RLAvatarUploadPrefix  = "rl:avatar_upload:"

	HeartbeatPrefix   = "heartbeat:"
	DeviceCountPrefix = "device_count:"

	LockContactPrefix = "lock:contact:"
	LockBlockPrefix   = "lock:block:"

	PrivacyCachePrefix = "privacy:"
)

const (
	TTLProfileUpdate = time.Hour
	TTLContactAdd    = time.Hour
	TTLBlockAction   = time.Hour
	TTLSearch        = time.Minute
	TTLReport        = 24 * time.Hour
	TTLStatusCreate  = 24 * time.Hour
	TTLStatusView    = 24 * time.Hour
	TTLAvatarUpload  = time.Hour

	TTLHeartbeat   = 35 * time.Second
	TTLDeviceCount = 35 * time.Second

	TTLLockContact = 10 * time.Second
	TTLLockBlock   = 10 * time.Second

	TTLPrivacyCache = 2 * time.Minute
)
