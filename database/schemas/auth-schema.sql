-- =====================================================
-- AUTH SCHEMA - Authentication & Session Management
-- =====================================================

-- Create Schema
CREATE SCHEMA IF NOT EXISTS auth;

-- Users Authentication Table
CREATE TABLE auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone_number VARCHAR(20) UNIQUE,
    phone_country_code VARCHAR(5),
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    password_hash TEXT NOT NULL,
    password_salt TEXT NOT NULL,
    password_algorithm VARCHAR(50) DEFAULT 'bcrypt',
    password_last_changed_at TIMESTAMPTZ,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret TEXT,
    two_factor_backup_codes TEXT[],
    account_status VARCHAR(50) DEFAULT 'active', -- active, suspended, banned, deleted, pending
    account_locked_until TIMESTAMPTZ,
    failed_login_attempts INTEGER DEFAULT 0,
    last_failed_login_at TIMESTAMPTZ,
    last_successful_login_at TIMESTAMPTZ,
    requires_password_change BOOLEAN DEFAULT FALSE,
    password_history JSONB DEFAULT '[]'::JSONB, -- Store last 5 password hashes
    security_questions JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    created_by_ip INET,
    created_by_user_agent TEXT
);

-- Sessions Table
CREATE TABLE auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    session_token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    device_id VARCHAR(255),
    device_name VARCHAR(255),
    device_type VARCHAR(50), -- mobile, tablet, desktop, web
    device_os VARCHAR(100),
    device_os_version VARCHAR(50),
    device_model VARCHAR(100),
    device_manufacturer VARCHAR(100),
    browser_name VARCHAR(100),
    browser_version VARCHAR(50),
    user_agent TEXT,
    ip_address INET NOT NULL,
    ip_country VARCHAR(100),
    ip_region VARCHAR(100),
    ip_city VARCHAR(100),
    ip_timezone VARCHAR(100),
    ip_isp VARCHAR(255),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    is_mobile BOOLEAN DEFAULT FALSE,
    is_trusted_device BOOLEAN DEFAULT FALSE,
    fcm_token TEXT, -- Firebase Cloud Messaging
    apns_token TEXT, -- Apple Push Notification
    push_enabled BOOLEAN DEFAULT TRUE,
    session_type VARCHAR(50) DEFAULT 'user', -- user, api, admin, service
    expires_at TIMESTAMPTZ NOT NULL,
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    last_refresh_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,
    parent_session_id UUID REFERENCES auth.sessions(id),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- OTP Verification Table
CREATE TABLE auth.otp_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    identifier VARCHAR(255) NOT NULL, -- email or phone
    identifier_type VARCHAR(20) NOT NULL, -- email, phone
    otp_code VARCHAR(10) NOT NULL,
    otp_hash TEXT NOT NULL,
    purpose VARCHAR(50) NOT NULL, -- registration, login, password_reset, phone_verify, email_verify, 2fa
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 5,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    sent_via VARCHAR(50), -- sms, email, voice
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}'::JSONB
);

-- OAuth Providers Table
CREATE TABLE auth.oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- google, facebook, apple, github
    provider_user_id VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    provider_username VARCHAR(255),
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    scope TEXT[],
    profile_data JSONB,
    is_primary BOOLEAN DEFAULT FALSE,
    linked_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    unlinked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);

-- Password Reset Tokens
CREATE TABLE auth.password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    email_sent_at TIMESTAMPTZ,
    email_opened_at TIMESTAMPTZ,
    link_clicked_at TIMESTAMPTZ
);

-- Email Verification Tokens
CREATE TABLE auth.email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token TEXT UNIQUE NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    attempts INTEGER DEFAULT 0
);

-- Security Events / Audit Log
CREATE TABLE auth.security_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    event_type VARCHAR(100) NOT NULL, -- login, logout, password_change, 2fa_enable, suspicious_activity
    event_category VARCHAR(50), -- authentication, authorization, account_management, security
    severity VARCHAR(20) DEFAULT 'info', -- info, warning, error, critical
    status VARCHAR(20), -- success, failure, blocked
    description TEXT,
    ip_address INET,
    user_agent TEXT,
    device_id VARCHAR(255),
    location_country VARCHAR(100),
    location_city VARCHAR(100),
    risk_score INTEGER, -- 0-100
    is_suspicious BOOLEAN DEFAULT FALSE,
    blocked_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Login History
CREATE TABLE auth.login_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    login_method VARCHAR(50), -- password, oauth, otp, biometric, api_key
    status VARCHAR(20), -- success, failed, blocked
    failure_reason TEXT,
    ip_address INET,
    user_agent TEXT,
    device_id VARCHAR(255),
    device_fingerprint TEXT,
    location_country VARCHAR(100),
    location_city VARCHAR(100),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    is_new_device BOOLEAN DEFAULT FALSE,
    is_new_location BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Rate Limiting Table
CREATE TABLE auth.rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier VARCHAR(255) NOT NULL, -- user_id, ip_address, api_key
    identifier_type VARCHAR(50) NOT NULL, -- user, ip, api_key, email
    action_type VARCHAR(100) NOT NULL, -- login, register, password_reset, api_call
    attempt_count INTEGER DEFAULT 1,
    window_start TIMESTAMPTZ DEFAULT NOW(),
    window_duration INTERVAL DEFAULT '1 hour',
    max_attempts INTEGER,
    blocked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(identifier, identifier_type, action_type)
);

-- Indexes for performance
CREATE INDEX idx_auth_users_email ON auth.users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_auth_users_phone ON auth.users(phone_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_auth_users_status ON auth.users(account_status);
CREATE INDEX idx_auth_sessions_user ON auth.sessions(user_id);
CREATE INDEX idx_auth_sessions_token ON auth.sessions(session_token);
CREATE INDEX idx_auth_sessions_device ON auth.sessions(device_id);
CREATE INDEX idx_auth_sessions_expires ON auth.sessions(expires_at);
CREATE INDEX idx_auth_otp_identifier ON auth.otp_verifications(identifier, identifier_type);
CREATE INDEX idx_auth_otp_expires ON auth.otp_verifications(expires_at);
CREATE INDEX idx_auth_security_events_user ON auth.security_events(user_id);
CREATE INDEX idx_auth_security_events_created ON auth.security_events(created_at);
CREATE INDEX idx_auth_login_history_user ON auth.login_history(user_id);
CREATE INDEX idx_auth_login_history_created ON auth.login_history(created_at);