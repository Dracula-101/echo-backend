CREATE OR REPLACE FUNCTION auth.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.update_device_fingerprint()
RETURNS TRIGGER AS $$
DECLARE
    v_device RECORD;
BEGIN
    SELECT device_id, device_name, device_type,
           device_os, device_os_version, device_model, device_manufacturer
    INTO v_device
    FROM auth.sessions
    WHERE id = NEW.session_id;

    NEW.device_fingerprint := auth.generate_device_fingerprint(
        v_device.device_id,
        v_device.device_name,
        v_device.device_type,
        v_device.device_os,
        v_device.device_os_version,
        v_device.device_model,
        v_device.device_manufacturer
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.log_password_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.password_hash IS DISTINCT FROM NEW.password_hash THEN
        PERFORM auth.log_security_event(
            NEW.id, NULL,
            'password_change', 'account_management', 'info', 'success',
            'User password was changed',
            NULL, NULL, NULL, NULL, NULL, '{}'::JSONB
        );
        NEW.password_last_changed_at := NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.log_2fa_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.two_factor_enabled IS DISTINCT FROM NEW.two_factor_enabled THEN
        PERFORM auth.log_security_event(
            NEW.id, NULL,
            CASE WHEN NEW.two_factor_enabled THEN '2fa_enable' ELSE '2fa_disable' END,
            'security', 'info', 'success',
            CASE WHEN NEW.two_factor_enabled
                THEN 'Two-factor authentication enabled'
                ELSE 'Two-factor authentication disabled'
            END,
            NULL, NULL, NULL, NULL, NULL, '{}'::JSONB
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.log_failed_login_attempt()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE auth.users
    SET failed_login_attempts = failed_login_attempts + 1,
        last_failed_login_at  = NOW()
    WHERE id = NEW.user_id;

    PERFORM auth.log_security_event(
        NEW.user_id, NULL,
        'failed_login', 'authentication', 'warning', 'failure',
        'Failed login attempt',
        NEW.ip_address, NEW.user_agent, NEW.device_id,
        NEW.location_country, NEW.location_city,
        jsonb_build_object('reason', NEW.failure_reason)
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.log_session_creation()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE auth.users
    SET last_successful_login_at = NOW(),
        failed_login_attempts    = 0
    WHERE id = NEW.user_id;

    PERFORM auth.log_security_event(
        NEW.user_id, NEW.id,
        'login', 'authentication', 'info', 'success',
        'User logged in successfully',
        NEW.ip_address, NEW.user_agent, NEW.device_id,
        NEW.ip_country, NEW.ip_city,
        jsonb_build_object(
            'is_new_device', NOT NEW.is_trusted_device,
            'device_type',   NEW.device_type
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.log_session_revocation()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.revoked_at IS NOT NULL AND OLD.revoked_at IS NULL THEN
        PERFORM auth.log_security_event(
            NEW.user_id, NEW.id,
            'logout', 'authentication', 'info', 'success',
            'Session revoked: ' || COALESCE(NEW.revoked_reason, 'user_logout'),
            NEW.ip_address, NEW.user_agent, NEW.device_id,
            NEW.ip_country, NEW.ip_city, '{}'::JSONB
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.prevent_user_deletion_with_active_sessions()
RETURNS TRIGGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM auth.sessions
    WHERE user_id   = OLD.id
      AND revoked_at IS NULL
      AND expires_at > NOW();

    IF v_count > 0 THEN
        RAISE EXCEPTION 'Cannot delete user with active sessions. Revoke all sessions first.';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.cleanup_user_oauth_providers()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM auth.oauth_providers WHERE user_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;