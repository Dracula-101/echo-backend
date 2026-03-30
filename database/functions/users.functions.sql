-- =====================================================
-- USERS SCHEMA — UTILITY FUNCTIONS
-- =====================================================
--
-- Description:  Callable utility functions for the users
--               schema. These are invoked by application
--               code or other SQL functions — NOT by
--               triggers.
--
-- Note:         Trigger handler functions (RETURNS TRIGGER)
--               live in users.trigger_functions.sql.
--
-- Dependencies: users schema tables must exist.
--               users.profiles, users.contacts,
--               users.blocked_users, users.status_history,
--               users.activity_log, users.contact_groups,
--               users.devices.
--
-- =====================================================


-- -------------------------------------------------
-- auth.current_user_id()   [RLS helper — referenced here
-- for documentation; defined in auth schema or via SET]
-- -------------------------------------------------


-- -------------------------------------------------
-- users.are_contacts(p_user_id, p_other_user_id)
-- -------------------------------------------------
-- Returns TRUE if p_other_user_id is an accepted
-- contact of p_user_id. Used by RLS policies and
-- application-level permission checks.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.are_contacts(p_user_id UUID, p_other_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users.contacts
        WHERE user_id = p_user_id
          AND contact_user_id = p_other_user_id
          AND status = 'accepted'
    );
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.is_blocked(p_user_id, p_other_user_id)
-- -------------------------------------------------
-- Returns TRUE if p_user_id has an active (un-unblocked)
-- block on p_other_user_id. Used by messaging validation
-- triggers and RLS policies.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.is_blocked(p_user_id UUID, p_other_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users.blocked_users
        WHERE user_id = p_user_id
          AND blocked_user_id = p_other_user_id
          AND unblocked_at IS NULL
    );
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.block_user(...)
-- -------------------------------------------------
-- Blocks p_blocked_user_id on behalf of p_user_id.
-- Also updates the contact record (if any) to
-- relationship_type = 'blocked'.
--
-- Raises an exception if the user is already blocked.
-- Returns the UUID of the new blocked_users row.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.block_user(
    p_user_id UUID,
    p_blocked_user_id UUID,
    p_block_reason TEXT DEFAULT NULL,
    p_block_type VARCHAR DEFAULT 'full'
)
RETURNS UUID AS $$
DECLARE
    v_block_id UUID;
BEGIN
    IF users.is_blocked(p_user_id, p_blocked_user_id) THEN
        RAISE EXCEPTION 'User is already blocked';
    END IF;

    INSERT INTO users.blocked_users (
        user_id, blocked_user_id, block_reason, block_type
    ) VALUES (
        p_user_id, p_blocked_user_id, p_block_reason, p_block_type
    ) RETURNING id INTO v_block_id;

    UPDATE users.contacts
    SET relationship_type = 'blocked',
        status = 'blocked',
        blocked_at = NOW(),
        block_reason = p_block_reason
    WHERE user_id = p_user_id
      AND contact_user_id = p_blocked_user_id;

    RETURN v_block_id;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.unblock_user(p_user_id, p_blocked_user_id)
-- -------------------------------------------------
-- Removes the active block. Optionally restores the
-- contact relationship to 'accepted' if it was
-- previously blocked.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.unblock_user(p_user_id UUID, p_blocked_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE users.blocked_users
    SET unblocked_at = NOW()
    WHERE user_id = p_user_id
      AND blocked_user_id = p_blocked_user_id
      AND unblocked_at IS NULL;

    UPDATE users.contacts
    SET relationship_type = 'contact',
        status = 'accepted'
    WHERE user_id = p_user_id
      AND contact_user_id = p_blocked_user_id
      AND relationship_type = 'blocked';
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.update_online_status(...)
-- -------------------------------------------------
-- Sets the user's online_status on their profile.
-- Called by the application when a WebSocket connection
-- opens, closes, or goes idle.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.update_online_status(
    p_user_id UUID,
    p_status VARCHAR,
    p_last_seen TIMESTAMPTZ DEFAULT NOW()
)
RETURNS VOID AS $$
BEGIN
    UPDATE users.profiles
    SET online_status = p_status,
        last_seen_at = p_last_seen,
        updated_at = NOW()
    WHERE user_id = p_user_id;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.set_inactive_users_offline(p_minutes)
-- -------------------------------------------------
-- Bulk-updates users who have been 'online' or 'away'
-- for longer than p_minutes to 'offline'. Intended to
-- be called by a periodic cleanup job.
--
-- Returns the number of profiles updated.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.set_inactive_users_offline(p_minutes INTEGER DEFAULT 5)
RETURNS INTEGER AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    UPDATE users.profiles
    SET online_status = 'offline'
    WHERE online_status IN ('online', 'away')
      AND last_seen_at < NOW() - (p_minutes || ' minutes')::INTERVAL;

    GET DIAGNOSTICS v_updated_count = ROW_COUNT;
    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.cleanup_expired_statuses()
-- -------------------------------------------------
-- Soft-deletes status_history rows whose expires_at
-- has passed. Called by a periodic cleanup job.
--
-- Returns the number of statuses expired.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.cleanup_expired_statuses()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    UPDATE users.status_history
    SET deleted_at = NOW()
    WHERE expires_at < NOW()
      AND deleted_at IS NULL;

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.get_user_contacts(...)
-- -------------------------------------------------
-- Returns a user's accepted contacts with profile info,
-- optionally filtered by relationship_type.
-- Results are ordered: pinned first, then favorites,
-- then alphabetical by display_name.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.get_user_contacts(
    p_user_id UUID,
    p_relationship_type VARCHAR DEFAULT NULL
)
RETURNS TABLE (
    contact_id UUID,
    username VARCHAR,
    display_name VARCHAR,
    avatar_url TEXT,
    online_status VARCHAR,
    last_seen_at TIMESTAMPTZ,
    is_favorite BOOLEAN,
    is_pinned BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.contact_user_id,
        p.username,
        p.display_name,
        p.avatar_url,
        p.online_status,
        p.last_seen_at,
        c.is_favorite,
        c.is_pinned
    FROM users.contacts c
    JOIN users.profiles p ON p.user_id = c.contact_user_id
    WHERE c.user_id = p_user_id
      AND c.status = 'accepted'
      AND (p_relationship_type IS NULL OR c.relationship_type = p_relationship_type)
    ORDER BY c.is_pinned DESC, c.is_favorite DESC, p.display_name ASC;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.search_users(p_search_term, p_limit)
-- -------------------------------------------------
-- Full-text + ILIKE search across usernames, display
-- names, and bios. Only returns users with
-- search_visibility = TRUE who are not deactivated.
-- Exact username matches rank highest, then verified
-- accounts.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.search_users(
    p_search_term TEXT,
    p_limit INTEGER DEFAULT 20
)
RETURNS TABLE (
    user_id UUID,
    username VARCHAR,
    display_name VARCHAR,
    avatar_url TEXT,
    bio TEXT,
    is_verified BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        p.user_id,
        p.username,
        p.display_name,
        p.avatar_url,
        p.bio,
        p.is_verified
    FROM users.profiles p
    WHERE p.search_visibility = TRUE
      AND p.deactivated_at IS NULL
      AND (
          p.username ILIKE '%' || p_search_term || '%'
          OR p.display_name ILIKE '%' || p_search_term || '%'
          OR to_tsvector('english', COALESCE(p.display_name, '') || ' ' || COALESCE(p.username, '') || ' ' || COALESCE(p.bio, ''))
             @@ plainto_tsquery('english', p_search_term)
      )
    ORDER BY
        CASE WHEN p.username = p_search_term THEN 0 ELSE 1 END,
        p.is_verified DESC,
        p.display_name
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.log_activity(...)
-- -------------------------------------------------
-- Inserts a row into users.activity_log. Called by
-- application code to record user actions with
-- optional before/after JSONB snapshots.
--
-- Returns the UUID of the new activity_log row.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.log_activity(
    p_user_id UUID,
    p_activity_type VARCHAR,
    p_activity_category VARCHAR,
    p_description TEXT,
    p_old_value JSONB DEFAULT NULL,
    p_new_value JSONB DEFAULT NULL,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_device_id VARCHAR DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_log_id UUID;
BEGIN
    INSERT INTO users.activity_log (
        user_id, activity_type, activity_category, description,
        old_value, new_value, ip_address, user_agent, device_id
    ) VALUES (
        p_user_id, p_activity_type, p_activity_category, p_description,
        p_old_value, p_new_value, p_ip_address, p_user_agent, p_device_id
    ) RETURNING id INTO v_log_id;

    RETURN v_log_id;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.update_contact_group_count(p_group_id)
-- -------------------------------------------------
-- Recalculates the member_count on a single
-- contact_groups row by counting how many contacts
-- reference this group in their contact_groups array.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.update_contact_group_count(p_group_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE users.contact_groups
    SET member_count = (
        SELECT COUNT(*)
        FROM users.contacts
        WHERE p_group_id = ANY(contact_groups)
          AND user_id = users.contact_groups.user_id
    )
    WHERE id = p_group_id;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- users.register_device(...)
-- -------------------------------------------------
-- Registers (or updates) a device for a user. Marks
-- the new device as is_current_device = TRUE and
-- demotes all other devices.
--
-- Uses UPSERT on (user_id, device_id) to handle
-- re-registration of an existing device.
--
-- Returns the UUID of the devices row.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION users.register_device(
    p_user_id UUID,
    p_device_id VARCHAR,
    p_device_name VARCHAR,
    p_device_type VARCHAR,
    p_device_model VARCHAR,
    p_device_manufacturer VARCHAR,
    p_os_name VARCHAR,
    p_os_version VARCHAR,
    p_app_version VARCHAR,
    p_fcm_token TEXT DEFAULT NULL,
    p_apns_token TEXT DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_device_record_id UUID;
BEGIN
    -- Demote all other devices
    UPDATE users.devices
    SET is_current_device = FALSE
    WHERE user_id = p_user_id;

    -- Insert or update the device
    INSERT INTO users.devices (
        user_id, device_id, device_name, device_type, device_model,
        device_manufacturer, os_name, os_version, app_version,
        fcm_token, apns_token, is_current_device, last_active_at
    ) VALUES (
        p_user_id, p_device_id, p_device_name, p_device_type, p_device_model,
        p_device_manufacturer, p_os_name, p_os_version, p_app_version,
        p_fcm_token, p_apns_token, TRUE, NOW()
    )
    ON CONFLICT (user_id, device_id) DO UPDATE SET
        device_name = EXCLUDED.device_name,
        device_type = EXCLUDED.device_type,
        is_current_device = TRUE,
        last_active_at = NOW(),
        fcm_token = COALESCE(EXCLUDED.fcm_token, users.devices.fcm_token),
        apns_token = COALESCE(EXCLUDED.apns_token, users.devices.apns_token),
        is_active = TRUE
    RETURNING id INTO v_device_record_id;

    RETURN v_device_record_id;
END;
$$ LANGUAGE plpgsql;
