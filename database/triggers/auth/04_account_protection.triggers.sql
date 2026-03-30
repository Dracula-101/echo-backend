-- =====================================================
-- AUTH TRIGGERS — Account Protection
-- =====================================================
--
-- Purpose:   Safeguard user account integrity:
--              - Prevent deletion when active sessions exist
--              - Clean up OAuth provider records on deletion
--
-- Depends:   functions/auth.trigger_functions.sql
--              -> auth.prevent_user_deletion_with_active_sessions()
--              -> auth.cleanup_user_oauth_providers()
--
-- Tables:    auth.users           (deletion guards)
--            auth.oauth_providers (cascade cleanup)
--
-- Note:      The deletion prevention trigger fires BEFORE
--            the cleanup trigger. If active sessions exist,
--            the deletion is aborted entirely.
--
-- =====================================================


-- Prevent deleting a user who has non-revoked, non-expired
-- sessions. Forces the caller to revoke all sessions first.
-- This avoids orphaned sessions and ensures a clean logout flow.
CREATE TRIGGER trigger_auth_users_prevent_deletion
    BEFORE DELETE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.prevent_user_deletion_with_active_sessions();

-- Explicitly remove OAuth provider records when a user is deleted.
-- While ON DELETE CASCADE handles this at the FK level, this trigger
-- provides a hook for additional cleanup logic if needed in the future.
CREATE TRIGGER trigger_auth_users_cleanup_oauth
    BEFORE DELETE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.cleanup_user_oauth_providers();
