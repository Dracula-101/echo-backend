package ratelimiter

import "time"

// ---------------- constants ----------------

const (
	RLLoginIPPrefix        = "rl:login:ip:"
	RLLoginUserPrefix      = "rl:login:user:"
	RLRegisterIPPrefix     = "rl:register:ip:"
	RLPwdForgotEmailPrefix = "rl:pwd_forgot:email:"
	RLPwdResetIPPrefix     = "rl:pwd_reset:ip:"
	RLResendVerifyPrefix   = "rl:resend_verify:user:"
	RLOtpSendPhonePrefix   = "rl:otp_send:phone:"
	RLRefreshDevicePrefix  = "rl:refresh:device:"
	RL2FAUserPrefix        = "rl:2fa:user:"

	LockLoginPrefix    = "lock:login:"
	LockRegisterPrefix = "lock:register:"

	PwdResetSentPrefix  = "pwd_reset_sent:"
	AccountLockedPrefix = "account_locked:"
	IPBlockedPrefix     = "ip_blocked:"
)

// ---------------- TTLs ----------------

var (
	TTLLoginWindow    = 15 * time.Minute
	TTLRegisterIP     = 1 * time.Hour
	TTLPwdForgotEmail = 1 * time.Hour
	TTLPwdResetIP     = 1 * time.Hour
	TTLResendVerify   = 1 * time.Hour
	TTLOtpSendPhone   = 10 * time.Minute
	TTLRefreshDevice  = 1 * time.Hour
	TTL2FAUser        = 1 * time.Hour

	TTLLockLogin    = 10 * time.Second
	TTLLockRegister = 10 * time.Second

	TTLPwdResetSent  = 5 * time.Minute
	TTLAccountLocked = 30 * time.Minute
	TTLIPBlocked     = 24 * time.Hour
)
