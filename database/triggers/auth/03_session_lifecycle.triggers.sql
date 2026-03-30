-- =====================================================
-- AUTH TRIGGERS — Session Lifecycle
-- =====================================================
--
-- Purpose:   Track the full lifecycle of user sessions:
--              - Login tracking (failed attempts counter)
--              - Successful login detection (counter reset)
--              - Device fingerprinting on login history
--              - Session creation logging (login event)
--              - Session revocation logging (logout event)
--
-- Depends:   functions/auth.trigger_functions.sql
--              -> auth.update_failed_login_attempts()
--              -> auth.update_last_successful_login()
--              -> auth.update_device_fingerprint()
--              -> auth.log_session_creation()
--              -> auth.log_session_revocation()
--
-- Tables:    auth.login_history  (login attempt tracking)
--            auth.sessions       (session create/revoke)
--
-- =====================================================


-- -----------------------------------------------
-- LOGIN ATTEMPT TRACKING
-- -----------------------------------------------

-- On every login attempt, update the user's failed login
-- counter: increment on failure, reset on success.
CREATE TRIGGER trigger_auth_users_failed_login_attempts
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_failed_login_attempts();

-- On successful login, reset failed_login_attempts to 0
-- and update last_successful_login_at on auth.users.
CREATE TRIGGER trigger_auth_sessions_successful_login
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_last_successful_login();

-- Generate a SHA-256 device fingerprint from session device
-- metadata and write it into the login_history row before persist.
CREATE TRIGGER trigger_auth_login_history_device_fingerprint
    BEFORE INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_device_fingerprint();


-- -----------------------------------------------
-- SESSION CREATION & REVOCATION
-- -----------------------------------------------

-- When a new session is created, record a successful login
-- in auth.login_history and log a 'login' security event.
CREATE TRIGGER trigger_auth_sessions_log_creation
    AFTER INSERT ON auth.sessions
    FOR EACH ROW
    EXECUTE FUNCTION auth.log_session_creation();

-- When a session's revoked_at transitions from NULL to non-NULL,
-- log a 'logout' security event with the revocation reason.
CREATE TRIGGER trigger_auth_sessions_log_revocation
    AFTER UPDATE ON auth.sessions
    FOR EACH ROW
    WHEN (NEW.revoked_at IS NOT NULL AND OLD.revoked_at IS NULL)
    EXECUTE FUNCTION auth.log_session_revocation();
