-- =====================================================
-- AUTH TRIGGERS — Security Event Logging
-- =====================================================
--
-- Purpose:   Log security-relevant changes to auth.security_events
--            for audit trail and anomaly detection. Covers:
--              - Password changes
--              - Two-factor authentication enable/disable
--              - Failed login attempts
--
-- Depends:   functions/auth.trigger_functions.sql
--              -> auth.log_password_change()
--              -> auth.log_2fa_change()
--              -> auth.log_failed_login_attempt()
--            functions/auth.functions.sql
--              -> auth.log_security_event()  (utility)
--
-- Tables:    auth.users          (password & 2FA changes)
--            auth.login_history  (failed login logging)
--
-- =====================================================


-- Log a security event and update password_last_changed_at
-- whenever a user's password_hash is modified.
-- Fires BEFORE UPDATE so it can mutate the NEW row.
CREATE TRIGGER trigger_auth_users_password_change
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    WHEN (OLD.password_hash IS DISTINCT FROM NEW.password_hash)
    EXECUTE FUNCTION auth.log_password_change();

-- Log a security event whenever two-factor authentication
-- is enabled or disabled on a user account.
CREATE TRIGGER trigger_auth_users_2fa_change
    AFTER UPDATE ON auth.users
    FOR EACH ROW
    WHEN (OLD.two_factor_enabled IS DISTINCT FROM NEW.two_factor_enabled)
    EXECUTE FUNCTION auth.log_2fa_change();

-- When a login attempt with status='failure' is recorded,
-- increment the user's failed_login_attempts counter and
-- write a warning-level security event with failure details.
CREATE TRIGGER trigger_auth_login_history_log_failed_attempts
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    WHEN (NEW.status = 'failure')
    EXECUTE FUNCTION auth.log_failed_login_attempt();
