-- =====================================================
-- NOTIFICATIONS SCHEMA - FUNCTIONS
-- =====================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION notifications.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to mark notification as read
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

-- Function to mark notification as seen
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

-- Function to mark all notifications as read for a user
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

-- Function to get unread count
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

-- Function to update user notification stats
CREATE OR REPLACE FUNCTION notifications.update_user_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO notifications.user_stats (
            user_id,
            total_notifications_sent,
            push_sent,
            last_notification_at
        ) VALUES (
            NEW.user_id,
            1,
            CASE WHEN NEW.platform IN ('ios', 'android') THEN 1 ELSE 0 END,
            NEW.created_at
        )
        ON CONFLICT (user_id) DO UPDATE SET
            total_notifications_sent = notifications.user_stats.total_notifications_sent + 1,
            push_sent = notifications.user_stats.push_sent + 
                CASE WHEN NEW.platform IN ('ios', 'android') THEN 1 ELSE 0 END,
            last_notification_at = NEW.created_at,
            updated_at = NOW();
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update delivery stats
CREATE OR REPLACE FUNCTION notifications.update_delivery_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'delivered' AND (OLD.status IS NULL OR OLD.status != 'delivered') THEN
        UPDATE notifications.user_stats
        SET total_notifications_delivered = total_notifications_delivered + 1,
            push_delivered = push_delivered + 1,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    ELSIF NEW.status = 'opened' AND (OLD.opened_at IS NULL) THEN
        UPDATE notifications.user_stats
        SET total_notifications_opened = total_notifications_opened + 1,
            push_opened = push_opened + 1,
            last_opened_notification_at = NEW.opened_at,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update email stats
CREATE OR REPLACE FUNCTION notifications.update_email_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'delivered' AND (OLD.status IS NULL OR OLD.status != 'delivered') THEN
        UPDATE notifications.user_stats
        SET email_delivered = email_delivered + 1,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    ELSIF NEW.opened_at IS NOT NULL AND OLD.opened_at IS NULL THEN
        UPDATE notifications.user_stats
        SET email_opened = email_opened + 1,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    ELSIF NEW.clicked_at IS NOT NULL AND OLD.clicked_at IS NULL THEN
        UPDATE notifications.user_stats
        SET email_clicked = email_clicked + 1,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up expired notifications
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

-- Function to check if user can receive notification
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
    
    IF v_prefs IS NULL THEN
        RETURN TRUE; -- Default to allowing notifications
    END IF;
    
    -- Check quiet hours
    IF v_prefs.quiet_hours_enabled THEN
        IF CURRENT_TIME BETWEEN v_prefs.quiet_hours_start AND v_prefs.quiet_hours_end THEN
            RETURN FALSE;
        END IF;
    END IF;
    
    -- Check global settings
    IF p_channel = 'push' AND NOT v_prefs.push_enabled THEN
        RETURN FALSE;
    ELSIF p_channel = 'email' AND NOT v_prefs.email_enabled THEN
        RETURN FALSE;
    ELSIF p_channel = 'sms' AND NOT v_prefs.sms_enabled THEN
        RETURN FALSE;
    END IF;
    
    -- Check specific notification type settings
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

-- Function to create notification
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

-- Function to update batch progress
CREATE OR REPLACE FUNCTION notifications.update_batch_progress()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.delivery_status = 'sent' AND (OLD.delivery_status IS NULL OR OLD.delivery_status != 'sent') THEN
        UPDATE notifications.batches
        SET sent_count = sent_count + 1
        WHERE id IN (
            SELECT jsonb_array_elements_text(metadata->'batch_id')::UUID
            FROM notifications.notifications
            WHERE id = NEW.id
        );
    ELSIF NEW.delivery_status = 'delivered' AND (OLD.delivery_status IS NULL OR OLD.delivery_status != 'delivered') THEN
        UPDATE notifications.batches
        SET delivered_count = delivered_count + 1
        WHERE id IN (
            SELECT jsonb_array_elements_text(metadata->'batch_id')::UUID
            FROM notifications.notifications
            WHERE id = NEW.id
        );
    ELSIF NEW.delivery_status = 'failed' AND (OLD.delivery_status IS NULL OR OLD.delivery_status != 'failed') THEN
        UPDATE notifications.batches
        SET failed_count = failed_count + 1
        WHERE id IN (
            SELECT jsonb_array_elements_text(metadata->'batch_id')::UUID
            FROM notifications.notifications
            WHERE id = NEW.id
        );
    END IF;
    
    IF NEW.is_read = TRUE AND OLD.is_read = FALSE THEN
        UPDATE notifications.batches
        SET opened_count = opened_count + 1
        WHERE id IN (
            SELECT jsonb_array_elements_text(metadata->'batch_id')::UUID
            FROM notifications.notifications
            WHERE id = NEW.id
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to increment announcement views
CREATE OR REPLACE FUNCTION notifications.increment_announcement_views()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE notifications.announcements
    SET view_count = view_count + 1
    WHERE id = NEW.announcement_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to increment announcement clicks
CREATE OR REPLACE FUNCTION notifications.increment_announcement_clicks()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.clicked = TRUE AND OLD.clicked = FALSE THEN
        UPDATE notifications.announcements
        SET click_count = click_count + 1
        WHERE id = NEW.announcement_id;
    END IF;
    
    IF NEW.dismissed = TRUE AND OLD.dismissed = FALSE THEN
        UPDATE notifications.announcements
        SET dismiss_count = dismiss_count + 1
        WHERE id = NEW.announcement_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;