-- Computes device_fingerprint by joining against auth.sessions before the row is inserted.
-- Must fire BEFORE INSERT so the computed value lands in the row.
CREATE TRIGGER trigger_auth_login_history_device_fingerprint
    BEFORE INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_device_fingerprint();

-- Increments failed_login_attempts on auth.users and writes a security event.
-- Fires only on failure rows — the WHEN clause is the canonical gate; the function
-- also checks status = 'failure' as a safety net.
-- NOTE: trigger_auth_users_failed_login_attempts from the original schema is removed.
--       It checked status = 'failed' (wrong value) and would have double-incremented
--       if fixed. A single trigger on the correct value is the right model.
CREATE TRIGGER trigger_auth_login_history_log_failed_attempts
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    WHEN (NEW.status = 'failure')
    EXECUTE FUNCTION auth.log_failed_login_attempt();