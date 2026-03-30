-- =====================================================
-- AUTH SCHEMA — TRIGGER HANDLER FUNCTIONS
-- =====================================================
--
-- Description:  Functions used exclusively as trigger handlers
--               for tables in the auth schema.
--
-- Dependencies: auth schema tables must exist.
--               auth.log_security_event() utility function
--               (defined in auth.functions.sql) is called by
--               several of these handlers.
--
-- Execution:    Load AFTER auth.functions.sql (utility functions)
--               and BEFORE any auth trigger files.
--
-- Naming:       All functions follow the pattern:
--               auth.<descriptive_action>()
--
-- =====================================================


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 1: TIMESTAMP MANAGEMENT
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- auth.update_updated_at_column()
-- -------------------------------------------------
-- Automatically sets the `updated_at` column to the
-- current timestamp whenever a row is modified.
-- Used by: auth.users, auth.sessions, auth.oauth_providers,
--          auth.api_keys
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 2: LOGIN ATTEMPT TRACKING
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- auth.update_failed_login_attempts()
-- -------------------------------------------------
-- Increments the failed_login_attempts counter and
-- records the timestamp on auth.users when a login
-- attempt with status = 'failed' is inserted into
-- auth.login_history.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.update_failed_login_attempts()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'failed' THEN
        UPDATE auth.users
        SET failed_login_attempts = failed_login_attempts + 1,
            last_failed_login_at = NOW()
        WHERE id = NEW.user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- auth.update_last_successful_login()
-- -------------------------------------------------
-- Resets the failed_login_attempts counter and updates
-- last_successful_login_at on auth.users when a login
-- attempt with status = 'success' is recorded.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.update_last_successful_login()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'success' THEN
        UPDATE auth.users
        SET last_successful_login_at = NOW(),
            failed_login_attempts = 0
        WHERE id = NEW.user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 3: DEVICE FINGERPRINTING
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- auth.update_device_fingerprint()
-- -------------------------------------------------
-- Generates a SHA-256 device fingerprint from session
-- device metadata and writes it into the login_history
-- row before it is persisted.
--
-- Depends on: auth.generate_device_fingerprint()
--             (defined in auth.functions.sql)
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.update_device_fingerprint()
RETURNS TRIGGER AS $$
BEGIN
    DECLARE
        v_device_info RECORD;
    BEGIN
        SELECT device_id, device_name, device_type,
               device_os, device_os_version,
               device_model, device_manufacturer
        INTO v_device_info
        FROM auth.sessions
        WHERE id = NEW.session_id;

        NEW.device_fingerprint = auth.generate_device_fingerprint(
            v_device_info.device_id,
            v_device_info.device_name,
            v_device_info.device_type,
            v_device_info.device_os,
            v_device_info.device_os_version,
            v_device_info.device_model,
            v_device_info.device_manufacturer
        );
    END;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 4: SECURITY EVENT LOGGING
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- auth.log_password_change()
-- -------------------------------------------------
-- Fires BEFORE UPDATE on auth.users when the
-- password_hash column changes. Logs a security event
-- and updates password_last_changed_at.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.log_password_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.password_hash IS DISTINCT FROM NEW.password_hash THEN
        PERFORM auth.log_security_event(
            NEW.id,
            NULL,
            'password_change',
            'account_management',
            'info',
            'success',
            'User password was changed',
            NULL,
            NULL,
            NULL,
            NULL,
            NULL,
            '{}'::JSONB
        );
        NEW.password_last_changed_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- auth.log_2fa_change()
-- -------------------------------------------------
-- Fires AFTER UPDATE on auth.users when the
-- two_factor_enabled flag changes. Logs a security
-- event indicating whether 2FA was enabled or disabled.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.log_2fa_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.two_factor_enabled IS DISTINCT FROM NEW.two_factor_enabled THEN
        PERFORM auth.log_security_event(
            NEW.id,
            NULL,
            CASE WHEN NEW.two_factor_enabled THEN '2fa_enable' ELSE '2fa_disable' END,
            'security',
            'info',
            'success',
            CASE WHEN NEW.two_factor_enabled
                 THEN 'Two-factor authentication enabled'
                 ELSE 'Two-factor authentication disabled'
            END,
            NULL,
            NULL,
            NULL,
            NULL,
            NULL,
            '{}'::JSONB
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- auth.log_failed_login_attempt()
-- -------------------------------------------------
-- Fires AFTER INSERT on auth.login_history when the
-- row has status = 'failure'. Increments the failed
-- login counter on auth.users and writes a security
-- event with the failure details.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.log_failed_login_attempt()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'failure' THEN
        UPDATE auth.users
        SET failed_login_attempts = failed_login_attempts + 1,
            last_failed_login_at = NOW()
        WHERE id = NEW.user_id;

        PERFORM auth.log_security_event(
            NEW.user_id,
            NULL,
            'failed_login',
            'authentication',
            'warning',
            'failure',
            'Failed login attempt',
            NEW.ip_address,
            NEW.user_agent,
            NEW.device_id,
            NEW.location_country,
            NEW.location_city,
            jsonb_build_object(
                'reason', NEW.failure_reason
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 5: SESSION LIFECYCLE
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- auth.log_session_creation()
-- -------------------------------------------------
-- Fires AFTER INSERT on auth.sessions. Performs three
-- tasks in sequence:
--   1. Resets failed_login_attempts on auth.users
--   2. Inserts a record into auth.login_history
--   3. Logs a 'login' security event with device info
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.log_session_creation()
RETURNS TRIGGER AS $$
DECLARE
    v_login_history_row RECORD;
BEGIN
    -- Reset failed attempts and record successful login
    UPDATE auth.users
    SET last_successful_login_at = NOW(),
        failed_login_attempts = 0
    WHERE id = NEW.user_id;

    -- Insert login history record
    INSERT INTO auth.login_history (
        user_id, session_id, login_method, status,
        ip_address, user_agent, device_id,
        location_country, location_city, latitude, longitude,
        is_new_device, is_new_location
    ) VALUES (
        NEW.user_id, NEW.id, 'password', 'success',
        NEW.ip_address, NEW.user_agent, NEW.device_id,
        NEW.ip_country, NEW.ip_city, NEW.latitude, NEW.longitude,
        NOT NEW.is_trusted_device, FALSE
    ) RETURNING * INTO v_login_history_row;

    -- Log the security event
    PERFORM auth.log_security_event(
        NEW.user_id,
        NEW.id,
        'login',
        'authentication',
        'info',
        'success',
        'User logged in successfully',
        NEW.ip_address,
        NEW.user_agent,
        NEW.device_id,
        NEW.ip_country,
        NEW.ip_city,
        jsonb_build_object(
            'is_new_device', NOT NEW.is_trusted_device,
            'device_type', NEW.device_type,
            'device_fingerprint', v_login_history_row.device_fingerprint
        )
    );

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- auth.log_session_revocation()
-- -------------------------------------------------
-- Fires AFTER UPDATE on auth.sessions when
-- revoked_at transitions from NULL to non-NULL.
-- Logs a 'logout' security event with the revocation
-- reason.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.log_session_revocation()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.revoked_at IS NOT NULL AND OLD.revoked_at IS NULL THEN
        PERFORM auth.log_security_event(
            NEW.user_id,
            NEW.id,
            'logout',
            'authentication',
            'info',
            'success',
            'Session revoked: ' || COALESCE(NEW.revoked_reason, 'User logged out'),
            NEW.ip_address,
            NEW.user_agent,
            NEW.device_id,
            NEW.ip_country,
            NEW.ip_city,
            jsonb_build_object()
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 6: ACCOUNT PROTECTION
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- auth.prevent_user_deletion_with_active_sessions()
-- -------------------------------------------------
-- Fires BEFORE DELETE on auth.users. Raises an
-- exception if the user has any non-revoked, non-expired
-- sessions, forcing the caller to revoke sessions first.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.prevent_user_deletion_with_active_sessions()
RETURNS TRIGGER AS $$
DECLARE
    v_active_sessions INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_active_sessions
    FROM auth.sessions
    WHERE user_id = OLD.id
      AND revoked_at IS NULL
      AND expires_at > NOW();

    IF v_active_sessions > 0 THEN
        RAISE EXCEPTION 'Cannot delete user with active sessions. Revoke all sessions first.';
    END IF;

    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- auth.cleanup_user_oauth_providers()
-- -------------------------------------------------
-- Fires BEFORE DELETE on auth.users. Removes all
-- associated OAuth provider records. This provides
-- explicit cleanup beyond the ON DELETE CASCADE
-- constraint for cases where cascading might be
-- deferred or additional cleanup logic is needed.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.cleanup_user_oauth_providers()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM auth.oauth_providers WHERE user_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
