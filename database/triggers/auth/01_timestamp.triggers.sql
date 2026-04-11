-- =====================================================
-- AUTH TRIGGERS — Timestamp Management
-- =====================================================
--
-- Purpose:   Automatically maintain `updated_at` columns
--            on auth schema tables whenever rows are modified.
--
-- Depends:   functions/auth.trigger_functions.sql
--              -> auth.update_updated_at_column()
--
-- Tables:    auth.users
--            auth.oauth_providers
--            auth.api_keys
--
-- =====================================================


-- Keep updated_at current on auth.users whenever any column is modified
CREATE TRIGGER trigger_auth_users_updated_at
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

-- Keep updated_at current on auth.oauth_providers (e.g. token refresh)
CREATE TRIGGER trigger_auth_oauth_updated_at
    BEFORE UPDATE ON auth.oauth_providers
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

-- Keep updated_at current on auth.api_keys (e.g. revocation, last_used)
CREATE TRIGGER trigger_auth_api_keys_updated_at
    BEFORE UPDATE ON auth.api_keys
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();
