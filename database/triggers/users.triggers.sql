-- =====================================================
-- USERS SCHEMA - TRIGGERS
-- =====================================================

-- Trigger to update updated_at on users.profiles
CREATE TRIGGER trigger_users_profiles_updated_at
    BEFORE UPDATE ON users.profiles
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Trigger to update updated_at on users.contacts
CREATE TRIGGER trigger_users_contacts_updated_at
    BEFORE UPDATE ON users.contacts
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Trigger to update updated_at on users.contact_groups
CREATE TRIGGER trigger_users_contact_groups_updated_at
    BEFORE UPDATE ON users.contact_groups
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Trigger to update updated_at on users.settings
CREATE TRIGGER trigger_users_settings_updated_at
    BEFORE UPDATE ON users.settings
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Trigger to update updated_at on users.privacy_overrides
CREATE TRIGGER trigger_users_privacy_overrides_updated_at
    BEFORE UPDATE ON users.privacy_overrides
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Trigger to update updated_at on users.preferences
CREATE TRIGGER trigger_users_preferences_updated_at
    BEFORE UPDATE ON users.preferences
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Trigger to update updated_at on users.reports
CREATE TRIGGER trigger_users_reports_updated_at
    BEFORE UPDATE ON users.reports
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Trigger to increment status views count
CREATE TRIGGER trigger_users_status_views_increment
    AFTER INSERT ON users.status_views
    FOR EACH ROW
    EXECUTE FUNCTION users.increment_status_views();

-- Trigger to create default profile when user is created
CREATE OR REPLACE FUNCTION users.create_default_profile()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users.profiles (user_id, username, display_name)
    VALUES (NEW.id, 'user_' || substring(NEW.id::text from 1 for 8), 'User');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_auth_users_create_profile
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION users.create_default_profile();

-- Trigger to create default settings when user is created
CREATE OR REPLACE FUNCTION users.create_default_settings()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users.settings (user_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_auth_users_create_settings
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION users.create_default_settings();

-- Trigger to log profile changes
CREATE OR REPLACE FUNCTION users.log_profile_changes()
RETURNS TRIGGER AS $$
DECLARE
    v_changes JSONB := '{}'::JSONB;
BEGIN
    IF OLD.display_name IS DISTINCT FROM NEW.display_name THEN
        v_changes := v_changes || jsonb_build_object('display_name', jsonb_build_object('old', OLD.display_name, 'new', NEW.display_name));
    END IF;
    
    IF OLD.bio IS DISTINCT FROM NEW.bio THEN
        v_changes := v_changes || jsonb_build_object('bio', jsonb_build_object('old', OLD.bio, 'new', NEW.bio));
    END IF;
    
    IF OLD.avatar_url IS DISTINCT FROM NEW.avatar_url THEN
        v_changes := v_changes || jsonb_build_object('avatar_url', jsonb_build_object('old', OLD.avatar_url, 'new', NEW.avatar_url));
    END IF;
    
    -- Check if v_changes is not an empty object
    IF v_changes != '{}'::JSONB THEN
        PERFORM users.log_activity(
            NEW.user_id,
            'profile_update',
            'profile',
            'User updated their profile',
            jsonb_build_object('old', v_changes),
            jsonb_build_object('new', v_changes),
            NULL, NULL, NULL
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_profiles_log_changes
    AFTER UPDATE ON users.profiles
    FOR EACH ROW
    WHEN (
        OLD.display_name IS DISTINCT FROM NEW.display_name
        OR OLD.bio IS DISTINCT FROM NEW.bio
        OR OLD.avatar_url IS DISTINCT FROM NEW.avatar_url
    )
    EXECUTE FUNCTION users.log_profile_changes();

-- Trigger to validate username format
CREATE OR REPLACE FUNCTION users.validate_username()
RETURNS TRIGGER AS $$
BEGIN
    -- Username must be alphanumeric with underscores, 3-50 chars
    IF NEW.username !~ '^[a-zA-Z0-9_]{3,50}$' THEN
        RAISE EXCEPTION 'Username must be 3-50 alphanumeric characters or underscores';
    END IF;
    
    -- Convert username to lowercase
    NEW.username := LOWER(NEW.username);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_profiles_validate_username
    BEFORE INSERT OR UPDATE ON users.profiles
    FOR EACH ROW
    WHEN (NEW.username IS NOT NULL)
    EXECUTE FUNCTION users.validate_username();

-- Trigger to prevent self-contact
CREATE OR REPLACE FUNCTION users.prevent_self_contact()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.user_id = NEW.contact_user_id THEN
        RAISE EXCEPTION 'Cannot add yourself as a contact';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_contacts_prevent_self
    BEFORE INSERT OR UPDATE ON users.contacts
    FOR EACH ROW
    EXECUTE FUNCTION users.prevent_self_contact();

-- Trigger to prevent self-blocking
CREATE OR REPLACE FUNCTION users.prevent_self_blocking()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.user_id = NEW.blocked_user_id THEN
        RAISE EXCEPTION 'Cannot block yourself';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_blocked_prevent_self
    BEFORE INSERT ON users.blocked_users
    FOR EACH ROW
    EXECUTE FUNCTION users.prevent_self_blocking();

-- Trigger to update contact accepted_at timestamp
CREATE OR REPLACE FUNCTION users.update_contact_accepted_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'accepted' AND OLD.status != 'accepted' THEN
        NEW.accepted_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_contacts_accepted
    BEFORE UPDATE ON users.contacts
    FOR EACH ROW
    WHEN (NEW.status = 'accepted' AND OLD.status != 'accepted')
    EXECUTE FUNCTION users.update_contact_accepted_at();

-- Trigger to set device last_active_at on update
CREATE OR REPLACE FUNCTION users.update_device_last_active()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_active_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_devices_last_active
    BEFORE UPDATE ON users.devices
    FOR EACH ROW
    EXECUTE FUNCTION users.update_device_last_active();

-- Trigger to clean up old devices
CREATE OR REPLACE FUNCTION users.cleanup_old_devices()
RETURNS TRIGGER AS $$
BEGIN
    -- Deactivate devices not used in 90 days
    UPDATE users.devices
    SET is_active = FALSE
    WHERE user_id = NEW.user_id
    AND last_active_at < NOW() - INTERVAL '90 days'
    AND is_active = TRUE;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_devices_cleanup
    AFTER INSERT ON users.devices
    FOR EACH ROW
    EXECUTE FUNCTION users.cleanup_old_devices();