-- =====================================================
-- AUTH SCHEMA - ROW LEVEL SECURITY (RLS)
-- =====================================================

-- Enable RLS on all auth tables
ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.otp_verifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.oauth_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.password_reset_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.email_verification_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.security_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.login_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.api_keys ENABLE ROW LEVEL SECURITY;

-- Helper function to get current user ID from JWT or session
CREATE OR REPLACE FUNCTION auth.current_user_id()
RETURNS UUID AS $$
BEGIN
    -- This should extract user_id from JWT token or session
    -- Implementation depends on your auth setup (Supabase, custom JWT, etc.)
    RETURN current_setting('app.current_user_id', TRUE)::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Helper function to check if user is admin
CREATE OR REPLACE FUNCTION auth.is_admin()
RETURNS BOOLEAN AS $$
BEGIN
    -- Check if current user has admin role
    RETURN EXISTS (
        SELECT 1 FROM auth.users
        WHERE id = auth.current_user_id()
        AND account_status = 'active'
        AND metadata->>'role' = 'admin'
    );
EXCEPTION
    WHEN OTHERS THEN
        RETURN FALSE;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- =====================================================
-- USERS TABLE POLICIES
-- =====================================================

-- Users can read their own record
DROP POLICY IF EXISTS users_select_own ON auth.users;
CREATE POLICY users_select_own
    ON auth.users
    FOR SELECT
    USING (id = auth.current_user_id());

-- Admins can read all users
DROP POLICY IF EXISTS users_select_admin ON auth.users;
CREATE POLICY users_select_admin
    ON auth.users
    FOR SELECT
    USING (auth.is_admin());

-- Users can update their own record (limited fields)
DROP POLICY IF EXISTS users_update_own ON auth.users;
CREATE POLICY users_update_own
    ON auth.users
    FOR UPDATE
    USING (id = auth.current_user_id())
    WITH CHECK (id = auth.current_user_id());

-- Admins can update any user
DROP POLICY IF EXISTS users_update_admin ON auth.users;
CREATE POLICY users_update_admin
    ON auth.users
    FOR UPDATE
    USING (auth.is_admin());

-- Service role can insert users (registration)
DROP POLICY IF EXISTS users_insert_service ON auth.users;
CREATE POLICY users_insert_service
    ON auth.users
    FOR INSERT
    WITH CHECK (TRUE); -- Should be restricted by service role

-- Admins can delete users
DROP POLICY IF EXISTS users_delete_admin ON auth.users;
CREATE POLICY users_delete_admin
    ON auth.users
    FOR DELETE
    USING (auth.is_admin());

-- =====================================================
-- SESSIONS TABLE POLICIES
-- =====================================================

-- Users can read their own sessions
DROP POLICY IF EXISTS sessions_select_own ON auth.sessions;
CREATE POLICY sessions_select_own
    ON auth.sessions
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can update their own sessions (for refresh)
DROP POLICY IF EXISTS sessions_update_own ON auth.sessions;
CREATE POLICY sessions_update_own
    ON auth.sessions
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- Service role can insert sessions
DROP POLICY IF EXISTS sessions_insert_service ON auth.sessions;
CREATE POLICY sessions_insert_service
    ON auth.sessions
    FOR INSERT
    WITH CHECK (TRUE);

-- Users can delete (revoke) their own sessions
DROP POLICY IF EXISTS sessions_delete_own ON auth.sessions;
CREATE POLICY sessions_delete_own
    ON auth.sessions
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Admins can manage all sessions
DROP POLICY IF EXISTS sessions_admin_all ON auth.sessions;
CREATE POLICY sessions_admin_all
    ON auth.sessions
    FOR ALL
    USING (auth.is_admin());

-- =====================================================
-- OTP VERIFICATIONS POLICIES
-- =====================================================

-- Service role can manage OTP (app layer handles validation)
DROP POLICY IF EXISTS otp_service_all ON auth.otp_verifications;
CREATE POLICY otp_service_all
    ON auth.otp_verifications
    FOR ALL
    USING (TRUE);

-- Users can read their own OTP records (for debugging)
DROP POLICY IF EXISTS otp_select_own ON auth.otp_verifications;
CREATE POLICY otp_select_own
    ON auth.otp_verifications
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR identifier IN (
            SELECT email FROM auth.users WHERE id = auth.current_user_id()
            UNION
            SELECT phone_number FROM auth.users WHERE id = auth.current_user_id()
        )
    );

-- =====================================================
-- OAUTH PROVIDERS POLICIES
-- =====================================================

-- Users can read their own OAuth connections
DROP POLICY IF EXISTS oauth_select_own ON auth.oauth_providers;
CREATE POLICY oauth_select_own
    ON auth.oauth_providers
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can delete their own OAuth connections
DROP POLICY IF EXISTS oauth_delete_own ON auth.oauth_providers;
CREATE POLICY oauth_delete_own
    ON auth.oauth_providers
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Service role can manage OAuth connections
DROP POLICY IF EXISTS oauth_service_all ON auth.oauth_providers;
CREATE POLICY oauth_service_all
    ON auth.oauth_providers
    FOR ALL
    USING (TRUE);

-- =====================================================
-- PASSWORD RESET TOKENS POLICIES
-- =====================================================

-- Service role manages password reset tokens
DROP POLICY IF EXISTS password_reset_service_all ON auth.password_reset_tokens;
CREATE POLICY password_reset_service_all
    ON auth.password_reset_tokens
    FOR ALL
    USING (TRUE);

-- Users can read their own tokens (limited)
DROP POLICY IF EXISTS password_reset_select_own ON auth.password_reset_tokens;
CREATE POLICY password_reset_select_own
    ON auth.password_reset_tokens
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- =====================================================
-- EMAIL VERIFICATION TOKENS POLICIES
-- =====================================================

-- Service role manages email verification tokens
DROP POLICY IF EXISTS email_verify_service_all ON auth.email_verification_tokens;
CREATE POLICY email_verify_service_all
    ON auth.email_verification_tokens
    FOR ALL
    USING (TRUE);

-- =====================================================
-- SECURITY EVENTS POLICIES
-- =====================================================

-- Users can read their own security events
DROP POLICY IF EXISTS security_events_select_own ON auth.security_events;
CREATE POLICY security_events_select_own
    ON auth.security_events
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Admins can read all security events
DROP POLICY IF EXISTS security_events_select_admin ON auth.security_events;
CREATE POLICY security_events_select_admin
    ON auth.security_events
    FOR SELECT
    USING (auth.is_admin());

-- Service role can insert security events
DROP POLICY IF EXISTS security_events_insert_service ON auth.security_events;
CREATE POLICY security_events_insert_service
    ON auth.security_events
    FOR INSERT
    WITH CHECK (TRUE);

-- =====================================================
-- LOGIN HISTORY POLICIES
-- =====================================================

-- Users can read their own login history
DROP POLICY IF EXISTS login_history_select_own ON auth.login_history;
CREATE POLICY login_history_select_own
    ON auth.login_history
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Admins can read all login history
DROP POLICY IF EXISTS login_history_select_admin ON auth.login_history;
CREATE POLICY login_history_select_admin
    ON auth.login_history
    FOR SELECT
    USING (auth.is_admin());

-- Service role can insert login history
DROP POLICY IF EXISTS login_history_insert_service ON auth.login_history;
CREATE POLICY login_history_insert_service
    ON auth.login_history
    FOR INSERT
    WITH CHECK (TRUE);

-- =====================================================
-- API KEYS POLICIES
-- =====================================================

-- Users can read their own API keys
DROP POLICY IF EXISTS api_keys_select_own ON auth.api_keys;
CREATE POLICY api_keys_select_own
    ON auth.api_keys
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can create their own API keys
DROP POLICY IF EXISTS api_keys_insert_own ON auth.api_keys;
CREATE POLICY api_keys_insert_own
    ON auth.api_keys
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

-- Users can update their own API keys (revoke, etc.)
DROP POLICY IF EXISTS api_keys_update_own ON auth.api_keys;
CREATE POLICY api_keys_update_own
    ON auth.api_keys
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- Users can delete their own API keys
DROP POLICY IF EXISTS api_keys_delete_own ON auth.api_keys;
CREATE POLICY api_keys_delete_own
    ON auth.api_keys
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Admins can manage all API keys
DROP POLICY IF EXISTS api_keys_admin_all ON auth.api_keys;
CREATE POLICY api_keys_admin_all
    ON auth.api_keys
    FOR ALL
    USING (auth.is_admin());

-- Service accounts can use their own API keys
DROP POLICY IF EXISTS api_keys_service_own ON auth.api_keys;
CREATE POLICY api_keys_service_own
    ON auth.api_keys
    FOR SELECT
    USING (service_name IS NOT NULL);