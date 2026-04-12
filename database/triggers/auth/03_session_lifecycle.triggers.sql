-- Resets failed_login_attempts to 0, stamps last_successful_login_at, and writes
-- a security event. Does NOT insert into login_history — auth-service owns that
-- insert directly so it can supply login_method and is_new_location, which the
-- trigger cannot know. The original schema had the trigger insert login_history,
-- which caused duplicate rows whenever auth-service also inserted. Removed.
CREATE TRIGGER trigger_auth_sessions_log_creation
    AFTER INSERT ON auth.sessions
    FOR EACH ROW
    EXECUTE FUNCTION auth.log_session_creation();

-- Writes a security event when a session transitions from active to revoked.
-- Fires only on the state transition (OLD.revoked_at IS NULL → NEW.revoked_at IS NOT NULL).
CREATE TRIGGER trigger_auth_sessions_log_revocation
    AFTER UPDATE ON auth.sessions
    FOR EACH ROW
    WHEN (NEW.revoked_at IS NOT NULL AND OLD.revoked_at IS NULL)
    EXECUTE FUNCTION auth.log_session_revocation();