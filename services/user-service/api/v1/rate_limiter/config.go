package ratelimiter

import "time"

const (
	RLLoginIPPrefix        = "rl:login:ip:"
	RLLoginUserPrefix      = "rl:login:user:"
	RLRegisterIPPrefix     = "rl:register:ip:"
	RLPwdForgotIPPrefix    = "rl:pwd_forgot:ip:"
	RLPwdForgotEmailPrefix = "rl:pwd_forgot:email:"
	RLPwdResetIPPrefix     = "rl:pwd_reset:ip:"
	RLResendVerifyPrefix   = "rl:resend_verify:"
	RLOtpSendPhonePrefix   = "rl:otp_send:phone:"
	RLRefreshDevicePrefix  = "rl:refresh:device:"
	RL2FAUserPrefix        = "rl:2fa:user:"

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

	LockLoginPrefix    = "lock:login:"
	LockRegisterPrefix = "lock:register:"
	LockContactPrefix  = "lock:contact:"
	LockBlockPrefix    = "lock:block:"

	IPBlockedPrefix     = "ip:blocked:"
	AccountLockedPrefix = "account:locked:"
	PwdResetSentPrefix  = "pwd_reset:sent:"

	PrivacyCachePrefix = "privacy:"
)

const (
	TTLLoginWindow    = 15 * time.Minute
	TTLRegisterIP     = time.Hour
	TTLPwdForgotIP    = time.Hour
	TTLPwdForgotEmail = time.Hour
	TTLPwdResetIP     = time.Hour
	TTLResendVerify   = time.Hour
	TTLOtpSendPhone   = time.Hour
	TTLRefreshDevice  = time.Hour
	TTL2FAUser        = 15 * time.Minute

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

	TTLLockLogin    = 10 * time.Second
	TTLLockRegister = 10 * time.Second
	TTLLockContact  = 10 * time.Second
	TTLLockBlock    = 10 * time.Second

	TTLIPBlocked             = 24 * time.Hour
	TTLAccountLocked         = 30 * time.Minute
	TTLAccountLockedExtended = 24 * time.Hour
	TTLPwdResetSent          = 5 * time.Minute

	TTLPrivacyCache = 2 * time.Minute
)
