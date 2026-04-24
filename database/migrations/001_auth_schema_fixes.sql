-- =====================================================
-- Migration 001 — Auth Schema Fixes
-- =====================================================
-- Reconciles the production DB state with the source files.
-- Safe to run multiple times (all statements are idempotent).
-- =====================================================

-- -------------------------------------------------------
-- 1. Drop the duplicate failed-login trigger and its function.
--    trigger_auth_users_failed_login_attempts fired on auth.login_history
--    but checked status = 'failed' (wrong value — DB stores 'failure').
--    It would also double-increment failed_login_attempts alongside
--    trigger_auth_login_history_log_failed_attempts. Removed entirely.
-- -------------------------------------------------------
DROP TRIGGER IF EXISTS trigger_auth_users_failed_login_attempts ON auth.login_history;
DROP FUNCTION IF EXISTS auth.update_failed_login_attempts();

-- -------------------------------------------------------
-- 2. Drop the trigger that duplicated login_history rows.
--    trigger_auth_sessions_successful_login fired on auth.sessions INSERT
--    and inserted into auth.login_history. auth-service now owns that
--    insert directly (it supplies login_method / is_new_location that a
--    trigger cannot know). The replacement trigger
--    trigger_auth_sessions_log_creation only resets failed_login_attempts
--    and logs a security event — it does NOT touch login_history.
-- -------------------------------------------------------
DROP TRIGGER IF EXISTS trigger_auth_sessions_successful_login ON auth.sessions;

-- -------------------------------------------------------
-- 3. Replace log_session_creation() to remove the login_history INSERT.
--    The old version in production inserts into auth.login_history on
--    every session creation, causing duplicate rows when auth-service
--    also inserts. The new version only updates auth.users and logs a
--    security event.
-- -------------------------------------------------------
CREATE OR REPLACE FUNCTION auth.log_session_creation()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE auth.users
    SET last_successful_login_at = NOW(),
        failed_login_attempts    = 0
    WHERE id = NEW.user_id;

    PERFORM auth.log_security_event(
        NEW.user_id, NEW.id,
        'login', 'authentication', 'info', 'success',
        'User logged in successfully',
        NEW.ip_address, NEW.user_agent, NEW.device_id,
        NEW.ip_country, NEW.ip_city,
        jsonb_build_object(
            'is_new_device', NOT NEW.is_trusted_device,
            'device_type',   NEW.device_type
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------
-- 4. Drop the plain token column from auth.password_reset_tokens.
--    The source schema uses only token_hash (bcrypt hash stored server-side).
--    Storing the raw token is a security risk. The column is removed; any
--    code that used it must use token_hash instead.
-- -------------------------------------------------------
ALTER TABLE auth.password_reset_tokens
    DROP COLUMN IF EXISTS token;

-- Fix the index that pointed to the now-dropped column.
-- The source already defines idx_auth_pwd_reset_token on token_hash.
DROP INDEX IF EXISTS auth.idx_auth_pwd_reset_token;

CREATE INDEX IF NOT EXISTS idx_auth_pwd_reset_token
    ON auth.password_reset_tokens(token_hash);

-- -------------------------------------------------------
-- 5. Add composite index on auth.sessions for efficient device lookup.
--    Used by auth-service to find the active session for a specific device,
--    e.g. during token refresh. Absent from production; already in source.
-- -------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_device
    ON auth.sessions(user_id, device_id)
    WHERE revoked_at IS NULL;
