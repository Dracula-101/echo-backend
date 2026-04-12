-- Blocks hard DELETE on auth.users if any active sessions exist.
-- Callers must revoke all sessions first (or use auth.soft_delete_user /
-- auth.hard_delete_user which handle revocation before the DELETE).
CREATE TRIGGER trigger_auth_users_prevent_deletion
    BEFORE DELETE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.prevent_user_deletion_with_active_sessions();

-- Removes all OAuth provider links before the user row is deleted.
-- Runs after the active-session guard so it only fires on legitimate deletes.
CREATE TRIGGER trigger_auth_users_cleanup_oauth
    BEFORE DELETE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.cleanup_user_oauth_providers();