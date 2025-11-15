-- =====================================================
-- AUTH SCHEMA - INDEXES
-- =====================================================

-- Users table indexes
CREATE INDEX idx_auth_users_email ON auth.users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_auth_users_phone ON auth.users(phone_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_auth_users_status ON auth.users(account_status);
CREATE INDEX idx_auth_users_created ON auth.users(created_at);
CREATE INDEX idx_auth_users_deleted ON auth.users(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_auth_users_2fa ON auth.users(two_factor_enabled) WHERE two_factor_enabled = TRUE;
CREATE INDEX idx_auth_users_locked ON auth.users(account_locked_until) WHERE account_locked_until IS NOT NULL;

-- Sessions table indexes
CREATE INDEX idx_auth_sessions_user ON auth.sessions(user_id);
CREATE INDEX idx_auth_sessions_token ON auth.sessions(session_token);
CREATE INDEX idx_auth_sessions_refresh_token ON auth.sessions(refresh_token);
CREATE INDEX idx_auth_sessions_device ON auth.sessions(device_id);
CREATE INDEX idx_auth_sessions_expires ON auth.sessions(expires_at);
CREATE INDEX idx_auth_sessions_active ON auth.sessions(user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX idx_auth_sessions_ip ON auth.sessions(ip_address);
CREATE INDEX idx_auth_sessions_created ON auth.sessions(created_at);
CREATE INDEX idx_auth_sessions_last_activity ON auth.sessions(last_activity_at);

-- OTP verifications indexes
CREATE INDEX idx_auth_otp_identifier ON auth.otp_verifications(identifier, identifier_type);
CREATE INDEX idx_auth_otp_user ON auth.otp_verifications(user_id);
CREATE INDEX idx_auth_otp_expires ON auth.otp_verifications(expires_at);
CREATE INDEX idx_auth_otp_purpose ON auth.otp_verifications(purpose);
CREATE INDEX idx_auth_otp_verified ON auth.otp_verifications(is_verified);
CREATE INDEX idx_auth_otp_created ON auth.otp_verifications(created_at);

-- OAuth providers indexes
CREATE INDEX idx_auth_oauth_user ON auth.oauth_providers(user_id);
CREATE INDEX idx_auth_oauth_provider ON auth.oauth_providers(provider);
CREATE INDEX idx_auth_oauth_provider_user ON auth.oauth_providers(provider, provider_user_id);
CREATE INDEX idx_auth_oauth_email ON auth.oauth_providers(provider_email);
CREATE INDEX idx_auth_oauth_primary ON auth.oauth_providers(user_id, is_primary) WHERE is_primary = TRUE;
CREATE INDEX idx_auth_oauth_linked ON auth.oauth_providers(linked_at);

-- Password reset tokens indexes
CREATE INDEX idx_auth_pwd_reset_user ON auth.password_reset_tokens(user_id);
CREATE INDEX idx_auth_pwd_reset_token ON auth.password_reset_tokens(token);
CREATE INDEX idx_auth_pwd_reset_expires ON auth.password_reset_tokens(expires_at);
CREATE INDEX idx_auth_pwd_reset_used ON auth.password_reset_tokens(used_at) WHERE used_at IS NULL;
CREATE INDEX idx_auth_pwd_reset_created ON auth.password_reset_tokens(created_at);

-- Email verification tokens indexes
CREATE INDEX idx_auth_email_verify_user ON auth.email_verification_tokens(user_id);
CREATE INDEX idx_auth_email_verify_email ON auth.email_verification_tokens(email);
CREATE INDEX idx_auth_email_verify_token ON auth.email_verification_tokens(token);
CREATE INDEX idx_auth_email_verify_expires ON auth.email_verification_tokens(expires_at);
CREATE INDEX idx_auth_email_verify_verified ON auth.email_verification_tokens(verified_at) WHERE verified_at IS NULL;

-- Security events indexes
CREATE INDEX idx_auth_security_events_user ON auth.security_events(user_id);
CREATE INDEX idx_auth_security_events_session ON auth.security_events(session_id);
CREATE INDEX idx_auth_security_events_type ON auth.security_events(event_type);
CREATE INDEX idx_auth_security_events_category ON auth.security_events(event_category);
CREATE INDEX idx_auth_security_events_severity ON auth.security_events(severity);
CREATE INDEX idx_auth_security_events_created ON auth.security_events(created_at);
CREATE INDEX idx_auth_security_events_suspicious ON auth.security_events(is_suspicious) WHERE is_suspicious = TRUE;
CREATE INDEX idx_auth_security_events_ip ON auth.security_events(ip_address);

-- Login history indexes
CREATE INDEX idx_auth_login_history_user ON auth.login_history(user_id);
CREATE INDEX idx_auth_login_history_session ON auth.login_history(session_id);
CREATE INDEX idx_auth_login_history_method ON auth.login_history(login_method);
CREATE INDEX idx_auth_login_history_status ON auth.login_history(status);
CREATE INDEX idx_auth_login_history_created ON auth.login_history(created_at);
CREATE INDEX idx_auth_login_history_ip ON auth.login_history(ip_address);
CREATE INDEX idx_auth_login_history_device ON auth.login_history(device_id);
CREATE INDEX idx_auth_login_history_new_device ON auth.login_history(user_id, is_new_device) WHERE is_new_device = TRUE;

-- Rate limits indexes
CREATE INDEX idx_auth_rate_limits_identifier ON auth.rate_limits(identifier, identifier_type, action_type);
CREATE INDEX idx_auth_rate_limits_action ON auth.rate_limits(action_type);
CREATE INDEX idx_auth_rate_limits_blocked ON auth.rate_limits(blocked_until) WHERE blocked_until IS NOT NULL;
CREATE INDEX idx_auth_rate_limits_window ON auth.rate_limits(window_start);

-- API keys indexes
CREATE INDEX idx_auth_api_keys_user ON auth.api_keys(user_id);
CREATE INDEX idx_auth_api_keys_hash ON auth.api_keys(key_hash);
CREATE INDEX idx_auth_api_keys_prefix ON auth.api_keys(key_prefix);
CREATE INDEX idx_auth_api_keys_active ON auth.api_keys(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_auth_api_keys_expires ON auth.api_keys(expires_at);
CREATE INDEX idx_auth_api_keys_last_used ON auth.api_keys(last_used_at);
CREATE INDEX idx_auth_api_keys_service ON auth.api_keys(service_name) WHERE service_name IS NOT NULL;