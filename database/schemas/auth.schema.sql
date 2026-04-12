CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE auth.users (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                       VARCHAR(255) UNIQUE NOT NULL,
    phone_number                VARCHAR(20) UNIQUE,
    phone_country_code          VARCHAR(5),
    email_verified              BOOLEAN DEFAULT FALSE,
    phone_verified              BOOLEAN DEFAULT FALSE,
    password_hash               TEXT NOT NULL,
    password_salt               TEXT NOT NULL,
    password_algorithm          VARCHAR(50) DEFAULT 'bcrypt',
    password_last_changed_at    TIMESTAMPTZ,
    two_factor_enabled          BOOLEAN DEFAULT FALSE,
    two_factor_secret           TEXT,
    two_factor_backup_codes     TEXT[],
    account_status              VARCHAR(50) DEFAULT 'pending_verification',
    account_locked_until        TIMESTAMPTZ,
    failed_login_attempts       INTEGER DEFAULT 0,
    last_failed_login_at        TIMESTAMPTZ,
    last_successful_login_at    TIMESTAMPTZ,
    requires_password_change    BOOLEAN DEFAULT FALSE,
    password_history            JSONB DEFAULT '[]'::JSONB,
    security_questions          JSONB,
    metadata                    JSONB DEFAULT '{}'::JSONB,
    created_at                  TIMESTAMPTZ DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ DEFAULT NOW(),
    deleted_at                  TIMESTAMPTZ,
    created_by_ip               INET,
    created_by_user_agent       TEXT
);

CREATE TABLE auth.sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    session_token       TEXT UNIQUE NOT NULL,
    refresh_token       TEXT UNIQUE,
    device_id           VARCHAR(255),
    device_name         VARCHAR(255),
    device_type         VARCHAR(50),
    device_os           VARCHAR(100),
    device_os_version   VARCHAR(50),
    device_model        VARCHAR(100),
    device_manufacturer VARCHAR(100),
    browser_name        VARCHAR(100),
    browser_version     VARCHAR(50),
    user_agent          TEXT,
    ip_address          INET NOT NULL,
    ip_country          VARCHAR(100),
    ip_region           VARCHAR(100),
    ip_city             VARCHAR(100),
    ip_timezone         VARCHAR(100),
    ip_isp              VARCHAR(255),
    latitude            DECIMAL(10, 8),
    longitude           DECIMAL(11, 8),
    is_mobile           BOOLEAN DEFAULT FALSE,
    is_trusted_device   BOOLEAN DEFAULT FALSE,
    fcm_token           TEXT,
    apns_token          TEXT,
    push_enabled        BOOLEAN DEFAULT TRUE,
    session_type        VARCHAR(50) DEFAULT 'user',
    expires_at          TIMESTAMPTZ NOT NULL,
    last_activity_at    TIMESTAMPTZ DEFAULT NOW(),
    last_refresh_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    revoked_at          TIMESTAMPTZ,
    revoked_reason      TEXT,
    metadata            JSONB DEFAULT '{}'::JSONB
);

CREATE TABLE auth.otp_verifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    identifier      VARCHAR(255) NOT NULL,
    identifier_type VARCHAR(20) NOT NULL,
    otp_code        VARCHAR(10) NOT NULL,
    otp_hash        TEXT NOT NULL,
    purpose         VARCHAR(50) NOT NULL,
    attempts        INTEGER DEFAULT 0,
    max_attempts    INTEGER DEFAULT 5,
    is_verified     BOOLEAN DEFAULT FALSE,
    verified_at     TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    sent_via        VARCHAR(50),
    ip_address      INET,
    user_agent      TEXT,
    metadata        JSONB DEFAULT '{}'::JSONB
);

CREATE TABLE auth.oauth_providers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider          VARCHAR(50) NOT NULL,
    provider_user_id  VARCHAR(255) NOT NULL,
    provider_email    VARCHAR(255),
    provider_username VARCHAR(255),
    access_token      TEXT,
    refresh_token     TEXT,
    token_expires_at  TIMESTAMPTZ,
    scope             TEXT[],
    profile_data      JSONB,
    is_primary        BOOLEAN DEFAULT FALSE,
    linked_at         TIMESTAMPTZ DEFAULT NOW(),
    last_used_at      TIMESTAMPTZ,
    unlinked_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);

CREATE TABLE auth.password_reset_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    attempts        INTEGER DEFAULT 0,
    max_attempts    INTEGER DEFAULT 5,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    ip_address      INET,
    user_agent      TEXT,
    email_sent_at   TIMESTAMPTZ,
    email_opened_at TIMESTAMPTZ,
    link_clicked_at TIMESTAMPTZ
);

CREATE TABLE auth.email_verification_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    email        VARCHAR(255) NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    verified_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    ip_address   INET,
    user_agent   TEXT,
    attempts     INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 5
);

CREATE TABLE auth.security_events (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id       UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    event_type       VARCHAR(100) NOT NULL,
    event_category   VARCHAR(50),
    severity         VARCHAR(20) DEFAULT 'info',
    status           VARCHAR(20),
    description      TEXT,
    ip_address       INET,
    user_agent       TEXT,
    device_id        VARCHAR(255),
    location_country VARCHAR(100),
    location_city    VARCHAR(100),
    risk_score       INTEGER,
    is_suspicious    BOOLEAN DEFAULT FALSE,
    blocked_reason   TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    metadata         JSONB DEFAULT '{}'::JSONB
);

CREATE TABLE auth.login_history (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id         UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    login_method       VARCHAR(50),
    status             VARCHAR(20) CHECK (status IN ('success', 'failure')),
    failure_reason     TEXT,
    ip_address         INET,
    user_agent         TEXT,
    device_id          VARCHAR(255),
    device_fingerprint TEXT,
    location_country   VARCHAR(100),
    location_city      VARCHAR(100),
    latitude           DECIMAL(10, 8),
    longitude          DECIMAL(11, 8),
    is_new_device      BOOLEAN DEFAULT FALSE,
    is_new_location    BOOLEAN DEFAULT FALSE,
    created_at         TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE auth.api_keys (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_name            VARCHAR(255) NOT NULL,
    key_hash            VARCHAR(255) NOT NULL UNIQUE,
    key_prefix          VARCHAR(20) NOT NULL,
    user_id             UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    service_name        VARCHAR(100),
    scopes              TEXT[],
    rate_limit_per_hour INTEGER DEFAULT 1000,
    is_active           BOOLEAN DEFAULT TRUE,
    expires_at          TIMESTAMPTZ,
    last_used_at        TIMESTAMPTZ,
    description         TEXT,
    metadata            JSONB,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE auth.outbox (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type   VARCHAR(100) NOT NULL,
    user_id      UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    failed_at    TIMESTAMPTZ,
    attempts     INTEGER DEFAULT 0,
    last_error   TEXT
);

COMMENT ON TABLE auth.users                     IS 'Core authentication table for user accounts';
COMMENT ON TABLE auth.sessions                  IS 'Active user sessions with device and location tracking';
COMMENT ON TABLE auth.otp_verifications         IS 'One-time password verification for 2FA and account verification';
COMMENT ON TABLE auth.oauth_providers           IS 'OAuth social login integrations';
COMMENT ON TABLE auth.password_reset_tokens     IS 'Password reset tokens — raw token never stored, lookup by token_hash only';
COMMENT ON TABLE auth.email_verification_tokens IS 'Email verification tokens — lookup by token_hash only';
COMMENT ON TABLE auth.security_events           IS 'Audit log for all security-related events';
COMMENT ON TABLE auth.login_history             IS 'Historical record of all login attempts, success and failure';
COMMENT ON TABLE auth.api_keys                  IS 'API keys for programmatic access';
COMMENT ON TABLE auth.outbox                    IS 'Transactional outbox for reliable Kafka event publishing via CDC';