-- =====================================================
-- AUTH SCHEMA — UTILITY FUNCTIONS
-- =====================================================
--
-- Description:  Callable utility functions for the auth
--               schema. These are invoked by application
--               code or other SQL functions — NOT by
--               triggers.
--
-- Note:         Trigger handler functions (RETURNS TRIGGER)
--               live in auth.trigger_functions.sql.
--
-- Dependencies: auth schema tables must exist.
--               pgcrypto extension (for digest/encode).
--               users.profiles, media.files (for soft_delete_user).
--               Multiple schemas (for hard_delete_user).
--
-- =====================================================


-- -------------------------------------------------
-- auth.generate_device_fingerprint(...)
-- -------------------------------------------------
-- Generates a deterministic SHA-256 fingerprint from
-- a device's identifying attributes. Used to detect
-- whether a login comes from a previously-seen device.
--
-- Parameters:
--   p_device_id           — unique device identifier
--   p_device_name         — user-facing device name
--   p_device_type         — phone / tablet / desktop
--   p_device_os           — iOS / Android / Windows / macOS
--   p_device_os_version   — e.g. "17.2"
--   p_device_model        — e.g. "iPhone 15 Pro"
--   p_device_manufacturer — e.g. "Apple"
--
-- Returns: hex-encoded SHA-256 string
-- -------------------------------------------------
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


-- -------------------------------------------------
-- auth.log_security_event(...)
-- -------------------------------------------------
-- Inserts a row into auth.security_events. Called by
-- trigger functions (log_password_change, log_2fa_change,
-- log_session_creation, etc.) and may also be called
-- directly from application code for custom events.
--
-- Parameters: see auth.security_events columns.
-- Returns:    UUID of the newly created event row.
-- -------------------------------------------------
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


-- -------------------------------------------------
-- auth.soft_delete_user(p_user_id)
-- -------------------------------------------------
-- Performs a reversible (soft) deletion of a user
-- account. The auth.users row is preserved with
-- account_status = 'deleted' and deleted_at set.
--
-- Actions taken:
--   1. Revoke all active sessions
--   2. Mark auth.users as deleted
--   3. Deactivate and anonymize the user's profile
--   4. Soft-delete the user's media files (scheduled
--      for permanent removal after 30 days)
--   5. Log a security event for the deletion
--
-- This does NOT remove data from other tables — the
-- user row remains so FK references stay valid. Use
-- auth.hard_delete_user() for full GDPR purge.
--
-- Returns TRUE on success, raises exception on failure.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.soft_delete_user(p_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_user RECORD;
    v_revoked_sessions INTEGER;
BEGIN
    -- -----------------------------------------------
    -- 1. Validate the user exists and is not already deleted
    -- -----------------------------------------------
    SELECT id, account_status, deleted_at
    INTO v_user
    FROM auth.users
    WHERE id = p_user_id;

    IF v_user IS NULL THEN
        RAISE EXCEPTION 'User % does not exist', p_user_id;
    END IF;

    IF v_user.account_status = 'deleted' OR v_user.deleted_at IS NOT NULL THEN
        RAISE EXCEPTION 'User % is already deleted', p_user_id;
    END IF;

    -- -----------------------------------------------
    -- 2. Revoke all active sessions
    -- -----------------------------------------------
    UPDATE auth.sessions
    SET revoked_at = NOW(),
        revoked_reason = 'account_deleted'
    WHERE user_id = p_user_id
      AND revoked_at IS NULL;

    GET DIAGNOSTICS v_revoked_sessions = ROW_COUNT;

    -- -----------------------------------------------
    -- 3. Mark the user as deleted
    -- -----------------------------------------------
    UPDATE auth.users
    SET account_status = 'deleted',
        deleted_at = NOW(),
        updated_at = NOW()
    WHERE id = p_user_id;

    -- -----------------------------------------------
    -- 4. Deactivate and anonymize the profile
    -- -----------------------------------------------
    -- Clear PII from the profile while preserving the
    -- row so conversations still show "Deleted User"
    UPDATE users.profiles
    SET deactivated_at = NOW(),
        online_status = 'offline',
        display_name = 'Deleted User',
        bio = NULL,
        avatar_url = NULL,
        cover_photo_url = NULL,
        website_url = NULL,
        location = NULL,
        search_visibility = FALSE,
        updated_at = NOW()
    WHERE user_id = p_user_id;

    -- -----------------------------------------------
    -- 5. Soft-delete the user's media files
    -- -----------------------------------------------
    -- Schedule files for permanent removal after 30 days
    -- (handled by media.cleanup_deleted_files() cron job)
    UPDATE media.files
    SET deleted_at = NOW(),
        permanently_delete_at = NOW() + INTERVAL '30 days'
    WHERE uploader_user_id = p_user_id
      AND deleted_at IS NULL;

    -- -----------------------------------------------
    -- 6. Log the deletion as a security event
    -- -----------------------------------------------
    PERFORM auth.log_security_event(
        p_user_id,
        NULL,                          -- no session (already revoked)
        'account_soft_delete',
        'account_management',
        'warning',
        'success',
        'User account soft-deleted. Sessions revoked: ' || v_revoked_sessions,
        NULL, NULL, NULL, NULL, NULL,
        jsonb_build_object(
            'deletion_type', 'soft',
            'sessions_revoked', v_revoked_sessions
        )
    );

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- auth.hard_delete_user(p_user_id)
-- -------------------------------------------------
-- Performs an irreversible (hard) deletion of a user
-- account. This permanently removes the auth.users
-- row and all associated data across all schemas.
--
-- Intended for GDPR "right to erasure" requests.
--
-- Actions taken:
--   1. Revoke all active sessions (required by the
--      prevent_user_deletion_with_active_sessions trigger)
--   2. Manually clean up 15+ tables that lack
--      ON DELETE CASCADE on their user FK
--   3. Log the deletion event (before removing user)
--   4. DELETE FROM auth.users — cascades handle 50+ tables
--
-- Tables with ON DELETE CASCADE are handled automatically.
-- Tables with ON DELETE SET NULL keep their rows but
-- lose the user reference (e.g. messages retain content
-- but sender_user_id becomes NULL).
--
-- Admin-created resources (templates, announcements,
-- batches, reports) are preserved with their
-- created_by / assigned_to fields nullified.
--
-- Returns TRUE on success, raises exception on failure.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION auth.hard_delete_user(p_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_user_exists BOOLEAN;
BEGIN
    -- -----------------------------------------------
    -- 1. Validate the user exists
    -- -----------------------------------------------
    SELECT EXISTS (
        SELECT 1 FROM auth.users WHERE id = p_user_id
    ) INTO v_user_exists;

    IF NOT v_user_exists THEN
        RAISE EXCEPTION 'User % does not exist', p_user_id;
    END IF;

    -- -----------------------------------------------
    -- 2. Revoke all active sessions
    -- -----------------------------------------------
    -- The BEFORE DELETE trigger
    -- auth.prevent_user_deletion_with_active_sessions()
    -- will block the DELETE if any active sessions remain.
    UPDATE auth.sessions
    SET revoked_at = NOW(),
        revoked_reason = 'account_hard_deleted'
    WHERE user_id = p_user_id
      AND revoked_at IS NULL;

    -- -----------------------------------------------
    -- 3. Log the deletion event BEFORE removing the user
    -- -----------------------------------------------
    -- This must happen before the DELETE because the
    -- security_events FK cascades on user deletion.
    -- The event will be deleted along with the user,
    -- but if security_events are replicated to an
    -- external audit log, this ensures the event is
    -- captured before cascade.
    PERFORM auth.log_security_event(
        p_user_id,
        NULL,
        'account_hard_delete',
        'account_management',
        'critical',
        'success',
        'User account permanently deleted (GDPR purge)',
        NULL, NULL, NULL, NULL, NULL,
        jsonb_build_object(
            'deletion_type', 'hard',
            'gdpr_purge', TRUE
        )
    );

    -- -----------------------------------------------
    -- 4. Clean up tables WITHOUT ON DELETE CASCADE
    -- -----------------------------------------------
    -- These tables have FK references to auth.users
    -- but no cascade rule, so they would cause FK
    -- violation errors if we just DELETE the user.

    -- === messages schema: user-specific data (DELETE) ===

    -- Message reactions posted by this user
    DELETE FROM messages.reactions
    WHERE user_id = p_user_id;

    -- Message delivery status records for this user
    DELETE FROM messages.delivery_status
    WHERE user_id = p_user_id;

    -- Poll votes cast by this user
    DELETE FROM messages.poll_votes
    WHERE user_id = p_user_id;

    -- Call participation records for this user
    DELETE FROM messages.call_participants
    WHERE user_id = p_user_id;

    -- Messages pinned by this user (unpin, not delete message)
    DELETE FROM messages.pinned_messages
    WHERE pinned_by_user_id = p_user_id;

    -- Message reports filed by this user
    DELETE FROM messages.message_reports
    WHERE reporter_user_id = p_user_id;

    -- Calls initiated by this user (nullify, keep call record)
    UPDATE messages.calls
    SET initiator_user_id = NULL
    WHERE initiator_user_id = p_user_id;

    -- === notifications schema: user-specific data (DELETE) ===

    -- Notification action responses from this user
    DELETE FROM notifications.action_responses
    WHERE user_id = p_user_id;

    -- Announcement views by this user
    DELETE FROM notifications.announcement_views
    WHERE user_id = p_user_id;

    -- === notifications schema: admin references (NULLIFY) ===
    -- Preserve admin-created resources but remove user link

    -- Announcements created by this user
    UPDATE notifications.announcements
    SET created_by_user_id = NULL
    WHERE created_by_user_id = p_user_id;

    -- Notification batches created by this user
    UPDATE notifications.batches
    SET created_by_user_id = NULL
    WHERE created_by_user_id = p_user_id;

    -- Notification templates created by this user
    UPDATE notifications.templates
    SET created_by_user_id = NULL
    WHERE created_by_user_id = p_user_id;

    -- === users schema: reports ===

    -- Reports filed ABOUT this user (remove, reporter stays)
    DELETE FROM users.reports
    WHERE reported_user_id = p_user_id;

    -- Reports assigned to this user for review (nullify)
    UPDATE users.reports
    SET assigned_to = NULL
    WHERE assigned_to = p_user_id;

    -- === analytics schema: admin references (NULLIFY) ===

    -- A/B tests created by this user
    UPDATE analytics.ab_tests
    SET created_by_user_id = NULL
    WHERE created_by_user_id = p_user_id;

    -- -----------------------------------------------
    -- 5. Delete the user — ON DELETE CASCADE handles
    --    the remaining 50+ tables automatically
    -- -----------------------------------------------
    DELETE FROM auth.users
    WHERE id = p_user_id;

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
