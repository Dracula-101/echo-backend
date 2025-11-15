-- =====================================================
-- NOTIFICATIONS SCHEMA - TRIGGERS
-- =====================================================

-- Trigger to update updated_at on notifications.notifications
CREATE TRIGGER trigger_notifications_updated_at
    BEFORE UPDATE ON notifications.notifications
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Trigger to update updated_at on notifications.user_preferences
CREATE TRIGGER trigger_user_preferences_updated_at
    BEFORE UPDATE ON notifications.user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Trigger to update updated_at on notifications.conversation_channels
CREATE TRIGGER trigger_conversation_channels_updated_at
    BEFORE UPDATE ON notifications.conversation_channels
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Trigger to update updated_at on notifications.templates
CREATE TRIGGER trigger_templates_updated_at
    BEFORE UPDATE ON notifications.templates
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Trigger to update updated_at on notifications.announcements
CREATE TRIGGER trigger_announcements_updated_at
    BEFORE UPDATE ON notifications.announcements
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Trigger to update updated_at on notifications.user_stats
CREATE TRIGGER trigger_user_stats_updated_at
    BEFORE UPDATE ON notifications.user_stats
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Trigger to update user notification stats on insert
CREATE TRIGGER trigger_notifications_update_stats_insert
    AFTER INSERT ON notifications.notifications
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_user_stats();

-- Trigger to update delivery stats
CREATE TRIGGER trigger_push_delivery_update_stats
    AFTER UPDATE ON notifications.push_delivery_log
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status OR OLD.opened_at IS DISTINCT FROM NEW.opened_at)
    EXECUTE FUNCTION notifications.update_delivery_stats();

-- Trigger to update email stats
CREATE TRIGGER trigger_email_notifications_update_stats
    AFTER UPDATE ON notifications.email_notifications
    FOR EACH ROW
    WHEN (
        OLD.status IS DISTINCT FROM NEW.status 
        OR OLD.opened_at IS DISTINCT FROM NEW.opened_at
        OR OLD.clicked_at IS DISTINCT FROM NEW.clicked_at
    )
    EXECUTE FUNCTION notifications.update_email_stats();

-- Trigger to update batch progress
CREATE TRIGGER trigger_notifications_batch_progress
    AFTER UPDATE ON notifications.notifications
    FOR EACH ROW
    WHEN (
        OLD.delivery_status IS DISTINCT FROM NEW.delivery_status
        OR OLD.is_read IS DISTINCT FROM NEW.is_read
    )
    EXECUTE FUNCTION notifications.update_batch_progress();

-- Trigger to increment announcement views
CREATE TRIGGER trigger_announcement_views_increment
    AFTER INSERT ON notifications.announcement_views
    FOR EACH ROW
    EXECUTE FUNCTION notifications.increment_announcement_views();

-- Trigger to increment announcement clicks/dismisses
CREATE TRIGGER trigger_announcement_views_clicks
    AFTER UPDATE ON notifications.announcement_views
    FOR EACH ROW
    WHEN (OLD.clicked IS DISTINCT FROM NEW.clicked OR OLD.dismissed IS DISTINCT FROM NEW.dismissed)
    EXECUTE FUNCTION notifications.increment_announcement_clicks();

-- Trigger to create default notification preferences
CREATE OR REPLACE FUNCTION notifications.create_default_preferences()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO notifications.user_preferences (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_auth_users_create_notification_preferences
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION notifications.create_default_preferences();

-- Trigger to create default notification stats
CREATE OR REPLACE FUNCTION notifications.create_default_notification_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO notifications.user_stats (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_auth_users_create_notification_stats
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION notifications.create_default_notification_stats();

-- Trigger to update notification delivery status when push is delivered
CREATE OR REPLACE FUNCTION notifications.update_notification_delivery_status()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'delivered' AND (OLD.status IS NULL OR OLD.status != 'delivered') THEN
        UPDATE notifications.notifications
        SET delivery_status = 'delivered',
            delivered_at = NEW.delivered_at
        WHERE id = NEW.notification_id;
    ELSIF NEW.status = 'failed' AND (OLD.status IS NULL OR OLD.status != 'failed') THEN
        UPDATE notifications.notifications
        SET delivery_status = 'failed',
            failed_reason = NEW.error_message
        WHERE id = NEW.notification_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_push_delivery_update_notification
    AFTER UPDATE ON notifications.push_delivery_log
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION notifications.update_notification_delivery_status();

-- Trigger to validate notification channel settings
CREATE OR REPLACE FUNCTION notifications.validate_conversation_channel()
RETURNS TRIGGER AS $$
BEGIN
    -- Ensure the conversation and user exist and user is participant
    IF NOT EXISTS (
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = NEW.conversation_id
        AND user_id = NEW.user_id
        AND left_at IS NULL
        AND removed_at IS NULL
    ) THEN
        RAISE EXCEPTION 'User % is not a participant of conversation %', NEW.user_id, NEW.conversation_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_conversation_channels_validate
    BEFORE INSERT OR UPDATE ON notifications.conversation_channels
    FOR EACH ROW
    EXECUTE FUNCTION notifications.validate_conversation_channel();