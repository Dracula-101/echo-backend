CREATE OR REPLACE FUNCTION auth.generate_device_fingerprint(
    p_device_id           TEXT,
    p_device_name         TEXT,
    p_device_type         TEXT,
    p_device_os           TEXT,
    p_device_os_version   TEXT,
    p_device_model        TEXT,
    p_device_manufacturer TEXT
)
RETURNS TEXT AS $$
DECLARE
    v_src TEXT;
BEGIN
    v_src := COALESCE(p_device_id, '')           || '|' ||
             COALESCE(p_device_name, '')          || '|' ||
             COALESCE(p_device_type, '')          || '|' ||
             COALESCE(p_device_os, '')            || '|' ||
             COALESCE(p_device_os_version, '')    || '|' ||
             COALESCE(p_device_model, '')         || '|' ||
             COALESCE(p_device_manufacturer, '');
    RETURN encode(digest(v_src, 'sha256'), 'hex');
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.log_security_event(
    p_user_id          UUID,
    p_session_id       UUID,
    p_event_type       VARCHAR,
    p_event_category   VARCHAR,
    p_severity         VARCHAR,
    p_status           VARCHAR,
    p_description      TEXT,
    p_ip_address       INET,
    p_user_agent       TEXT,
    p_device_id        TEXT,
    p_location_country VARCHAR,
    p_location_city    VARCHAR,
    p_metadata         JSONB
)
RETURNS UUID AS $$
DECLARE
    v_id UUID;
BEGIN
    INSERT INTO auth.security_events (
        user_id, session_id, event_type, event_category,
        severity, status, description, ip_address,
        user_agent, device_id, location_country, location_city, metadata
    ) VALUES (
        p_user_id, p_session_id, p_event_type, p_event_category,
        p_severity, p_status, p_description, p_ip_address,
        p_user_agent, p_device_id, p_location_country, p_location_city, p_metadata
    ) RETURNING id INTO v_id;
    RETURN v_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.soft_delete_user(p_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_user             RECORD;
    v_revoked_sessions INTEGER;
BEGIN
    SELECT id, account_status, deleted_at INTO v_user
    FROM auth.users WHERE id = p_user_id;

    IF v_user IS NULL THEN
        RAISE EXCEPTION 'User % does not exist', p_user_id;
    END IF;
    IF v_user.account_status = 'deleted' OR v_user.deleted_at IS NOT NULL THEN
        RAISE EXCEPTION 'User % is already deleted', p_user_id;
    END IF;

    UPDATE auth.sessions
    SET revoked_at     = NOW(),
        revoked_reason = 'account_deleted'
    WHERE user_id = p_user_id AND revoked_at IS NULL;
    GET DIAGNOSTICS v_revoked_sessions = ROW_COUNT;

    UPDATE auth.users
    SET account_status = 'deleted',
        deleted_at     = NOW(),
        updated_at     = NOW()
    WHERE id = p_user_id;

    UPDATE users.profiles
    SET deactivated_at    = NOW(),
        online_status     = 'offline',
        display_name      = 'Deleted User',
        bio               = NULL,
        avatar_url        = NULL,
        cover_image_url   = NULL,
        website_url       = NULL,
        search_visibility = FALSE,
        updated_at        = NOW()
    WHERE user_id = p_user_id;

    UPDATE media.files
    SET deleted_at            = NOW(),
        permanently_delete_at = NOW() + INTERVAL '30 days'
    WHERE uploader_user_id = p_user_id AND deleted_at IS NULL;

    PERFORM auth.log_security_event(
        p_user_id, NULL,
        'account_soft_delete', 'account_management', 'warning', 'success',
        'User account soft-deleted. Sessions revoked: ' || v_revoked_sessions,
        NULL, NULL, NULL, NULL, NULL,
        jsonb_build_object('deletion_type', 'soft', 'sessions_revoked', v_revoked_sessions)
    );
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION auth.hard_delete_user(p_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM auth.users WHERE id = p_user_id) THEN
        RAISE EXCEPTION 'User % does not exist', p_user_id;
    END IF;

    UPDATE auth.sessions
    SET revoked_at     = NOW(),
        revoked_reason = 'account_hard_deleted'
    WHERE user_id = p_user_id AND revoked_at IS NULL;

    PERFORM auth.log_security_event(
        p_user_id, NULL,
        'account_hard_delete', 'account_management', 'critical', 'success',
        'User account permanently deleted (GDPR purge)',
        NULL, NULL, NULL, NULL, NULL,
        jsonb_build_object('deletion_type', 'hard', 'gdpr_purge', TRUE)
    );

    DELETE FROM messages.reactions         WHERE user_id = p_user_id;
    DELETE FROM messages.delivery_status   WHERE user_id = p_user_id;
    DELETE FROM messages.poll_votes        WHERE user_id = p_user_id;
    DELETE FROM messages.call_participants WHERE user_id = p_user_id;
    DELETE FROM messages.pinned_messages   WHERE pinned_by_user_id = p_user_id;
    DELETE FROM messages.message_reports   WHERE reporter_user_id = p_user_id;
    UPDATE messages.calls
        SET initiator_user_id = NULL WHERE initiator_user_id = p_user_id;

    DELETE FROM notifications.action_responses   WHERE user_id = p_user_id;
    DELETE FROM notifications.announcement_views WHERE user_id = p_user_id;
    UPDATE notifications.announcements
        SET created_by_user_id = NULL WHERE created_by_user_id = p_user_id;
    UPDATE notifications.batches
        SET created_by_user_id = NULL WHERE created_by_user_id = p_user_id;
    UPDATE notifications.templates
        SET created_by_user_id = NULL WHERE created_by_user_id = p_user_id;

    DELETE FROM users.reports WHERE reported_user_id = p_user_id;
    UPDATE users.reports SET assigned_to = NULL WHERE assigned_to = p_user_id;

    UPDATE analytics.ab_tests
        SET created_by_user_id = NULL WHERE created_by_user_id = p_user_id;

    DELETE FROM auth.users WHERE id = p_user_id;
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;