-- =====================================================
-- NOTIFICATIONS SCHEMA — UTILITY FUNCTIONS
-- =====================================================
--
-- Description:  Callable utility functions for the
--               notifications schema. These are invoked
--               by application code or periodic cleanup
--               jobs — NOT directly by triggers.
--
-- Note:         Trigger handler functions (RETURNS TRIGGER)
--               live in notifications.trigger_functions.sql.
--
-- Dependencies: notifications schema tables must exist.
--
-- =====================================================


-- -------------------------------------------------
-- notifications.mark_as_read(p_notification_id, p_user_id)
-- -------------------------------------------------
-- Marks a single notification as read for the given
-- user. No-ops if the notification is already read or
-- does not belong to the user.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.mark_as_read(p_notification_id UUID, p_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE notifications.notifications
    SET is_read = TRUE,
        read_at = NOW()
    WHERE id = p_notification_id
      AND user_id = p_user_id
      AND is_read = FALSE;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- notifications.mark_as_seen(p_notification_id, p_user_id)
-- -------------------------------------------------
-- Marks a single notification as seen (appeared in the
-- notification tray but not necessarily opened). No-ops
-- if already seen.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.mark_as_seen(p_notification_id UUID, p_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE notifications.notifications
    SET is_seen = TRUE,
        seen_at = NOW()
    WHERE id = p_notification_id
      AND user_id = p_user_id
      AND is_seen = FALSE;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- notifications.mark_all_as_read(p_user_id)
-- -------------------------------------------------
-- Marks ALL unread, non-deleted notifications for the
-- user as read. Returns the number of notifications
-- updated.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.mark_all_as_read(p_user_id UUID)
RETURNS INTEGER AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    UPDATE notifications.notifications
    SET is_read = TRUE,
        read_at = NOW()
    WHERE user_id = p_user_id
      AND is_read = FALSE
      AND deleted_at IS NULL;

    GET DIAGNOSTICS v_updated_count = ROW_COUNT;
    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- notifications.get_unread_count(p_user_id)
-- -------------------------------------------------
-- Returns the total number of unread, non-deleted,
-- non-expired notifications for the user. Used by
-- the application to display badge counts.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.get_unread_count(p_user_id UUID)
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM notifications.notifications
    WHERE user_id = p_user_id
      AND is_read = FALSE
      AND deleted_at IS NULL
      AND (expires_at IS NULL OR expires_at > NOW());

    RETURN v_count;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- notifications.cleanup_expired_notifications()
-- -------------------------------------------------
-- Soft-deletes notifications whose expires_at has
-- passed. Intended to be called by a periodic
-- cleanup job.
--
-- Returns the number of notifications expired.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.cleanup_expired_notifications()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    UPDATE notifications.notifications
    SET deleted_at = NOW()
    WHERE expires_at < NOW()
      AND deleted_at IS NULL;

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- notifications.can_receive_notification(...)
-- -------------------------------------------------
-- Checks whether a user can receive a notification of
-- the given type on the specified channel, based on
-- their user_preferences. Evaluates:
--   1. Quiet hours (time-based suppression)
--   2. Global channel toggle (push/email/sms)
--   3. Per-type channel preference
--
-- Returns TRUE if the notification should be sent,
-- FALSE if suppressed.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.can_receive_notification(
    p_user_id UUID,
    p_notification_type VARCHAR,
    p_channel VARCHAR -- 'push', 'email', 'sms'
)
RETURNS BOOLEAN AS $$
DECLARE
    v_prefs RECORD;
    v_can_receive BOOLEAN := FALSE;
BEGIN
    SELECT * INTO v_prefs
    FROM notifications.user_preferences
    WHERE user_id = p_user_id;

    -- No preferences row → default to allowing
    IF v_prefs IS NULL THEN
        RETURN TRUE;
    END IF;

    -- Quiet hours check
    IF v_prefs.quiet_hours_enabled THEN
        IF CURRENT_TIME BETWEEN v_prefs.quiet_hours_start AND v_prefs.quiet_hours_end THEN
            RETURN FALSE;
        END IF;
    END IF;

    -- Global channel toggle
    IF p_channel = 'push' AND NOT v_prefs.push_enabled THEN
        RETURN FALSE;
    ELSIF p_channel = 'email' AND NOT v_prefs.email_enabled THEN
        RETURN FALSE;
    ELSIF p_channel = 'sms' AND NOT v_prefs.sms_enabled THEN
        RETURN FALSE;
    END IF;

    -- Per-type preferences
    IF p_notification_type = 'message' THEN
        v_can_receive := CASE p_channel
            WHEN 'push' THEN v_prefs.message_push
            WHEN 'email' THEN v_prefs.message_email
            WHEN 'sms' THEN v_prefs.message_sms
            ELSE FALSE
        END;
    ELSIF p_notification_type = 'mention' THEN
        v_can_receive := CASE p_channel
            WHEN 'push' THEN v_prefs.mention_push
            WHEN 'email' THEN v_prefs.mention_email
            WHEN 'sms' THEN v_prefs.mention_sms
            ELSE FALSE
        END;
    ELSIF p_notification_type = 'call' THEN
        v_can_receive := CASE p_channel
            WHEN 'push' THEN v_prefs.call_push
            WHEN 'email' THEN v_prefs.call_email
            WHEN 'sms' THEN v_prefs.call_sms
            ELSE FALSE
        END;
    ELSE
        v_can_receive := TRUE; -- Default for unknown types
    END IF;

    RETURN v_can_receive;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- notifications.create_notification(...)
-- -------------------------------------------------
-- Creates a new notification for p_user_id. This is
-- the primary entry point for sending notifications
-- from application code. The AFTER INSERT trigger on
-- notifications.notifications will automatically
-- update user_stats.
--
-- Returns the UUID of the new notification.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.create_notification(
    p_user_id UUID,
    p_notification_type VARCHAR,
    p_title VARCHAR,
    p_body TEXT,
    p_related_user_id UUID DEFAULT NULL,
    p_related_message_id UUID DEFAULT NULL,
    p_related_conversation_id UUID DEFAULT NULL,
    p_action_url TEXT DEFAULT NULL,
    p_priority VARCHAR DEFAULT 'normal'
)
RETURNS UUID AS $$
DECLARE
    v_notification_id UUID;
BEGIN
    INSERT INTO notifications.notifications (
        user_id, notification_type, notification_category,
        title, body,
        related_user_id, related_message_id, related_conversation_id,
        action_url, priority
    ) VALUES (
        p_user_id, p_notification_type, 'social',
        p_title, p_body,
        p_related_user_id, p_related_message_id, p_related_conversation_id,
        p_action_url, p_priority
    ) RETURNING id INTO v_notification_id;

    RETURN v_notification_id;
END;
$$ LANGUAGE plpgsql;
