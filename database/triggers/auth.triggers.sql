-- =====================================================
-- AUTH SCHEMA - TRIGGERS
-- =====================================================

-- Trigger to update updated_at on auth.users
CREATE TRIGGER trigger_auth_users_updated_at
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

-- Trigger to update failed_login_attempts on auth.users
CREATE TRIGGER trigger_auth_users_failed_login_attempts
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_failed_login_attempts();

-- Trigger to log failed login attempts
CREATE TRIGGER trigger_auth_login_history_log_failed_attempts
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    WHEN (NEW.status = 'failure')
    EXECUTE FUNCTION auth.log_failed_login_attempt();

-- Trigger to update last_successful_login_at when session is created
CREATE TRIGGER trigger_auth_sessions_successful_login
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_last_successful_login();

-- Trigger to update device fingerprint on auth.login_history
CREATE TRIGGER trigger_auth_login_history_device_fingerprint
    BEFORE INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_device_fingerprint();


-- Trigger to update updated_at on auth.sessions
CREATE TRIGGER trigger_auth_sessions_updated_at
    BEFORE UPDATE ON auth.sessions
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

-- Trigger to update updated_at on auth.oauth_providers
CREATE TRIGGER trigger_auth_oauth_updated_at
    BEFORE UPDATE ON auth.oauth_providers
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

-- Trigger to update updated_at on auth.api_keys
CREATE TRIGGER trigger_auth_api_keys_updated_at
    BEFORE UPDATE ON auth.api_keys
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

CREATE TRIGGER trigger_auth_users_password_change
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    WHEN (OLD.password_hash IS DISTINCT FROM NEW.password_hash)
    EXECUTE FUNCTION auth.log_password_change();

CREATE TRIGGER trigger_auth_users_2fa_change
    AFTER UPDATE ON auth.users
    FOR EACH ROW
    WHEN (OLD.two_factor_enabled IS DISTINCT FROM NEW.two_factor_enabled)
    EXECUTE FUNCTION auth.log_2fa_change();

CREATE TRIGGER trigger_auth_sessions_log_creation
    AFTER INSERT ON auth.sessions
    FOR EACH ROW
    EXECUTE FUNCTION auth.log_session_creation();

CREATE TRIGGER trigger_auth_sessions_log_revocation
    AFTER UPDATE ON auth.sessions
    FOR EACH ROW
    WHEN (NEW.revoked_at IS NOT NULL AND OLD.revoked_at IS NULL)
    EXECUTE FUNCTION auth.log_session_revocation();

CREATE TRIGGER trigger_auth_users_prevent_deletion
    BEFORE DELETE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.prevent_user_deletion_with_active_sessions();

-- Trigger to clean up related OAuth providers on user deletion
CREATE OR REPLACE FUNCTION auth.cleanup_user_oauth_providers()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM auth.oauth_providers WHERE user_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;