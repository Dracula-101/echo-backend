CREATE OR REPLACE FUNCTION notifications.mark_as_read(p_notification_id UUID, p_user_id UUID) RETURNS VOID AS $$
BEGIN
    UPDATE notifications.notifications
    SET is_read = TRUE, read_at = NOW()
    WHERE id = p_notification_id AND user_id = p_user_id AND is_read = FALSE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION notifications.mark_as_seen(p_notification_id UUID, p_user_id UUID) RETURNS VOID AS $$
BEGIN
    UPDATE notifications.notifications
    SET is_seen = TRUE, seen_at = NOW()
    WHERE id = p_notification_id AND user_id = p_user_id AND is_seen = FALSE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION notifications.mark_all_as_read(p_user_id UUID) RETURNS INTEGER AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    UPDATE notifications.notifications
    SET is_read = TRUE, read_at = NOW()
    WHERE user_id = p_user_id AND is_read = FALSE AND deleted_at IS NULL;
    GET DIAGNOSTICS v_updated_count = ROW_COUNT;
    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION notifications.get_unread_count(p_user_id UUID) RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM notifications.notifications
    WHERE user_id    = p_user_id
      AND is_read    = FALSE
      AND deleted_at IS NULL
      AND (expires_at IS NULL OR expires_at > NOW());
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION notifications.cleanup_expired_notifications() RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    UPDATE notifications.notifications
    SET deleted_at = NOW()
    WHERE expires_at < NOW() AND deleted_at IS NULL;
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION notifications.can_receive_notification(
    p_user_id UUID, p_notification_type VARCHAR, p_channel VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    v_prefs       RECORD;
    v_can_receive BOOLEAN := FALSE;
BEGIN
    SELECT * INTO v_prefs FROM notifications.user_preferences WHERE user_id = p_user_id;
    IF v_prefs IS NULL THEN RETURN TRUE; END IF;

    IF v_prefs.quiet_hours_enabled THEN
        IF CURRENT_TIME BETWEEN v_prefs.quiet_hours_start AND v_prefs.quiet_hours_end THEN
            RETURN FALSE;
        END IF;
    END IF;

    IF p_channel = 'push'  AND NOT v_prefs.push_enabled  THEN RETURN FALSE; END IF;
    IF p_channel = 'email' AND NOT v_prefs.email_enabled THEN RETURN FALSE; END IF;
    IF p_channel = 'sms'   AND NOT v_prefs.sms_enabled   THEN RETURN FALSE; END IF;

    IF p_notification_type = 'message' THEN
        v_can_receive := CASE p_channel
            WHEN 'push' THEN v_prefs.message_push WHEN 'email' THEN v_prefs.message_email
            WHEN 'sms'  THEN v_prefs.message_sms  ELSE FALSE END;
    ELSIF p_notification_type = 'mention' THEN
        v_can_receive := CASE p_channel
            WHEN 'push' THEN v_prefs.mention_push WHEN 'email' THEN v_prefs.mention_email
            WHEN 'sms'  THEN v_prefs.mention_sms  ELSE FALSE END;
    ELSIF p_notification_type = 'call' THEN
        v_can_receive := CASE p_channel
            WHEN 'push' THEN v_prefs.call_push WHEN 'email' THEN v_prefs.call_email
            WHEN 'sms'  THEN v_prefs.call_sms  ELSE FALSE END;
    ELSE
        v_can_receive := TRUE;
    END IF;

    RETURN v_can_receive;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION notifications.create_notification(
    p_user_id UUID, p_notification_type VARCHAR, p_title VARCHAR, p_body TEXT,
    p_related_user_id UUID DEFAULT NULL, p_related_message_id UUID DEFAULT NULL,
    p_related_conversation_id UUID DEFAULT NULL, p_action_url TEXT DEFAULT NULL,
    p_priority VARCHAR DEFAULT 'normal'
) RETURNS UUID AS $$
DECLARE
    v_notification_id UUID;
BEGIN
    INSERT INTO notifications.notifications (
        user_id, notification_type, notification_category, title, body,
        related_user_id, related_message_id, related_conversation_id,
        action_url, priority
    ) VALUES (
        p_user_id, p_notification_type, 'social', p_title, p_body,
        p_related_user_id, p_related_message_id, p_related_conversation_id,
        p_action_url, p_priority
    ) RETURNING id INTO v_notification_id;
    RETURN v_notification_id;
END;
$$ LANGUAGE plpgsql;
