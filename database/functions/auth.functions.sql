-- =====================================================
-- AUTH SCHEMA - FUNCTIONS
-- =====================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION auth.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update last_failed_login_at and increment failed_login_attempts
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

-- Function to update last_successful_login_at when session is created
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

-- Function to create device fingerprint
CREATE OR REPLACE FUNCTION auth.generate_device_fingerprint(
    p_device_id TEXT,
    p_device_name TEXT,
    p_device_type TEXT,
    p_device_os TEXT,
    p_device_os_version TEXT,
    p_device_model TEXT,
    p_device_manufacturer TEXT
)
RETURNS TEXT
AS $$
DECLARE
    v_fingerprint_source TEXT;
    v_fingerprint TEXT;
BEGIN
    v_fingerprint_source := COALESCE(p_device_id, '') || '|' ||
                            COALESCE(p_device_name, '') || '|' ||
                            COALESCE(p_device_type, '') || '|' ||
                            COALESCE(p_device_os, '') || '|' ||
                            COALESCE(p_device_os_version, '') || '|' ||
                            COALESCE(p_device_model, '') || '|' ||  
                            COALESCE(p_device_manufacturer, '');
    v_fingerprint := encode(digest(v_fingerprint_source, 'sha256'), 'hex');
    RETURN v_fingerprint;
END;
$$ LANGUAGE plpgsql;


-- Function to update device fingerprint
CREATE OR REPLACE FUNCTION auth.update_device_fingerprint()
RETURNS TRIGGER AS $$
BEGIN
    -- get the device info from auth.sessions
    DECLARE
        v_device_info RECORD;
    BEGIN
        SELECT device_id, device_name, device_type, device_os, device_os_version, device_model, device_manufacturer
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

-- Function to log security events
CREATE OR REPLACE FUNCTION auth.log_security_event(
    p_user_id UUID,
    p_session_id UUID,
    p_event_type VARCHAR,
    p_event_category VARCHAR,
    p_severity VARCHAR,
    p_status VARCHAR,
    p_description TEXT,
    p_ip_address INET,
    p_user_agent TEXT,
    p_device_id TEXT,
    p_location_country VARCHAR,
    p_location_city VARCHAR,
    p_metadata JSONB
)
RETURNS UUID AS $$
DECLARE
    v_event_id UUID;
BEGIN
    INSERT INTO auth.security_events (
        user_id, session_id, event_type, event_category,
        severity, status, description, ip_address,
        user_agent, device_id, location_country, location_city, metadata
    ) VALUES (
        p_user_id, p_session_id, p_event_type, p_event_category,
        p_severity, p_status, p_description, p_ip_address,
        p_user_agent, p_device_id, p_location_country, p_location_city, p_metadata
    ) RETURNING id INTO v_event_id;
    
    RETURN v_event_id;
END;
$$ LANGUAGE plpgsql;

-- Trigger to log security event on password change
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

-- Trigger to log security event on 2FA enable/disable
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
            CASE WHEN NEW.two_factor_enabled THEN 'Two-factor authentication enabled' ELSE 'Two-factor authentication disabled' END,
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

-- Trigger to log failed login attempts
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
                'reason', NEW.failure_reason,
                'device_fingerprint', NEW.device_fingerprint
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to log session creation
CREATE OR REPLACE FUNCTION auth.log_session_creation()
RETURNS TRIGGER AS $$
DECLARE
    v_login_history_row RECORD;
BEGIN
    UPDATE auth.users
    SET last_successful_login_at = NOW(),
        failed_login_attempts = 0
    WHERE id = NEW.user_id;

    -- insert into login_history and get the inserted row back
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
        -- include inserted login_history row as JSONB along with existing metadata
        jsonb_build_object(
            'is_new_device', NOT NEW.is_trusted_device,
            'device_type', NEW.device_type,
            'device_fingerprint', v_login_history_row.device_fingerprint
        )
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;



-- Trigger to log session revocation
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

-- Trigger to prevent user deletion if active sessions exist
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

-- Trigger to clean up related OAuth providers on user deletion
CREATE OR REPLACE FUNCTION auth.cleanup_user_oauth_providers()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM auth.oauth_providers WHERE user_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;