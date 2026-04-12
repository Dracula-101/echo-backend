-- Writes a security event and stamps password_last_changed_at when the password
-- hash changes. Fires BEFORE UPDATE so it can mutate NEW.password_last_changed_at
-- in the same write without a second UPDATE.
CREATE TRIGGER trigger_auth_users_password_change
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    WHEN (OLD.password_hash IS DISTINCT FROM NEW.password_hash)
    EXECUTE FUNCTION auth.log_password_change();

-- Writes a security event when two_factor_enabled toggles in either direction.
-- Fires AFTER UPDATE because it only reads NEW/OLD — no need to mutate the row.
CREATE TRIGGER trigger_auth_users_2fa_change
    AFTER UPDATE ON auth.users
    FOR EACH ROW
    WHEN (OLD.two_factor_enabled IS DISTINCT FROM NEW.two_factor_enabled)
    EXECUTE FUNCTION auth.log_2fa_change();