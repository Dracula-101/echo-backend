-- =====================================================
-- USERS SCHEMA — TRIGGER HANDLER FUNCTIONS
-- =====================================================
--
-- Description:  Functions used exclusively as trigger handlers
--               for tables in the users schema.
--
-- Dependencies: users schema tables must exist.
--               auth.users table must exist (cross-schema
--               triggers fire on auth.users INSERT).
--               users.log_activity() utility function
--               (defined in users.functions.sql).
--
-- Execution:    Load AFTER users.functions.sql (utility functions)
--               and BEFORE any users trigger files.
--
-- =====================================================


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 1: TIMESTAMP MANAGEMENT
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- users.update_updated_at_column()
-- -------------------------------------------------
-- Automatically sets the `updated_at` column to the
-- current timestamp whenever a row is modified.
-- Used by: users.profiles, users.contacts,
--          users.contact_groups, users.settings,
--          users.privacy_overrides, users.preferences,
--          users.reports
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 2: USER REGISTRATION SETUP
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- These functions fire on auth.users INSERT to create
-- the default rows a new user needs in the users schema.

-- -------------------------------------------------
-- users.create_default_profile()
-- -------------------------------------------------
-- Creates a starter profile in users.profiles when a
-- new row is inserted into auth.users. Generates a
-- default username from the first 8 chars of the UUID.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.create_default_profile()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users.profiles (user_id, username, display_name)
    VALUES (NEW.id, 'user_' || substring(NEW.id::text from 1 for 8), 'User');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- users.create_default_device()
-- -------------------------------------------------
-- Creates a placeholder device record in users.devices
-- when a new row is inserted into auth.users. The real
-- device info is registered via users.register_device()
-- on first login.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.create_default_device()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users.devices (user_id, device_id, device_name, is_active, last_active_at)
    VALUES (NEW.id, 'default_device', 'Default Device', TRUE, NOW());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- users.create_default_settings()
-- -------------------------------------------------
-- Creates a default settings row in users.settings
-- when a new row is inserted into auth.users. All
-- settings columns use their DEFAULT values.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.create_default_settings()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users.settings (user_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 3: INPUT VALIDATION
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- users.validate_username()
-- -------------------------------------------------
-- Fires BEFORE INSERT or UPDATE on users.profiles.
-- Enforces username rules:
--   - Must be 3-50 characters
--   - Only alphanumeric and underscores allowed
--   - Automatically lowercased
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.validate_username()
RETURNS TRIGGER AS $$
BEGIN
    -- Username must be alphanumeric with underscores, 3-50 chars
    IF NEW.username !~ '^[a-zA-Z0-9_]{3,50}$' THEN
        RAISE EXCEPTION 'Username must be 3-50 alphanumeric characters or underscores';
    END IF;

    -- Normalize to lowercase
    NEW.username := LOWER(NEW.username);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- users.prevent_self_contact()
-- -------------------------------------------------
-- Fires BEFORE INSERT or UPDATE on users.contacts.
-- Prevents a user from adding themselves as a contact.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.prevent_self_contact()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.user_id = NEW.contact_user_id THEN
        RAISE EXCEPTION 'Cannot add yourself as a contact';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- users.prevent_self_blocking()
-- -------------------------------------------------
-- Fires BEFORE INSERT on users.blocked_users.
-- Prevents a user from blocking themselves.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.prevent_self_blocking()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.user_id = NEW.blocked_user_id THEN
        RAISE EXCEPTION 'Cannot block yourself';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 4: PROFILE & CONTACT TRACKING
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- users.log_profile_changes()
-- -------------------------------------------------
-- Fires AFTER UPDATE on users.profiles when key fields
-- (display_name, bio, avatar_url) change. Builds a
-- JSON diff and writes it to users.activity_log via
-- the users.log_activity() utility function.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.log_profile_changes()
RETURNS TRIGGER AS $$
DECLARE
    v_changes JSONB := '{}'::JSONB;
BEGIN
    IF OLD.display_name IS DISTINCT FROM NEW.display_name THEN
        v_changes := v_changes || jsonb_build_object(
            'display_name', jsonb_build_object('old', OLD.display_name, 'new', NEW.display_name)
        );
    END IF;

    IF OLD.bio IS DISTINCT FROM NEW.bio THEN
        v_changes := v_changes || jsonb_build_object(
            'bio', jsonb_build_object('old', OLD.bio, 'new', NEW.bio)
        );
    END IF;

    IF OLD.avatar_url IS DISTINCT FROM NEW.avatar_url THEN
        v_changes := v_changes || jsonb_build_object(
            'avatar_url', jsonb_build_object('old', OLD.avatar_url, 'new', NEW.avatar_url)
        );
    END IF;

    -- Only log if at least one tracked field changed
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

-- -------------------------------------------------
-- users.update_contact_accepted_at()
-- -------------------------------------------------
-- Fires BEFORE UPDATE on users.contacts when the
-- status transitions to 'accepted'. Sets the
-- accepted_at timestamp.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.update_contact_accepted_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'accepted' AND OLD.status != 'accepted' THEN
        NEW.accepted_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- users.update_contact_interaction()
-- -------------------------------------------------
-- Updates the last_interaction_at timestamp and
-- increments the interaction_count on users.contacts.
-- Intended to be called from application-level triggers
-- or directly when a meaningful interaction occurs.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.update_contact_interaction()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users.contacts
    SET last_interaction_at = NOW(),
        interaction_count = interaction_count + 1
    WHERE user_id = NEW.user_id
      AND contact_user_id = NEW.contact_user_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 5: STATUS & ACTIVITY
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- users.increment_status_views()
-- -------------------------------------------------
-- Fires AFTER INSERT on users.status_views.
-- Increments the views_count counter on the
-- associated users.status_history row.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.increment_status_views()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users.status_history
    SET views_count = views_count + 1
    WHERE id = NEW.status_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 6: DEVICE MANAGEMENT
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- users.update_device_last_active()
-- -------------------------------------------------
-- Fires BEFORE UPDATE on users.devices. Automatically
-- bumps the last_active_at timestamp to NOW() on
-- every device update.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.update_device_last_active()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_active_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- users.cleanup_old_devices()
-- -------------------------------------------------
-- Fires AFTER INSERT on users.devices. When a new
-- device is registered, deactivates any of the same
-- user's devices that have not been used in 90 days.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.cleanup_old_devices()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users.devices
    SET is_active = FALSE
    WHERE user_id = NEW.user_id
      AND last_active_at < NOW() - INTERVAL '90 days'
      AND is_active = TRUE;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
