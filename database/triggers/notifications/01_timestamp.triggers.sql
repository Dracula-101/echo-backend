-- =====================================================
-- NOTIFICATIONS TRIGGERS — Timestamp Management
-- =====================================================
--
-- Purpose:   Automatically maintain `updated_at` columns
--            on notifications schema tables whenever rows
--            are modified.
--
-- Depends:   functions/notifications.trigger_functions.sql
--              -> notifications.update_updated_at_column()
--
-- Tables:    notifications.notifications
--            notifications.user_preferences
--            notifications.conversation_channels
--            notifications.templates
--            notifications.announcements
--            notifications.user_stats
--
-- =====================================================


-- Keep updated_at current on notifications
CREATE TRIGGER trigger_notifications_updated_at
    BEFORE UPDATE ON notifications.notifications
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Keep updated_at current on user notification preferences
CREATE TRIGGER trigger_user_preferences_updated_at
    BEFORE UPDATE ON notifications.user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Keep updated_at current on per-conversation channel settings
CREATE TRIGGER trigger_conversation_channels_updated_at
    BEFORE UPDATE ON notifications.conversation_channels
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Keep updated_at current on notification templates
CREATE TRIGGER trigger_templates_updated_at
    BEFORE UPDATE ON notifications.templates
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Keep updated_at current on in-app announcements
CREATE TRIGGER trigger_announcements_updated_at
    BEFORE UPDATE ON notifications.announcements
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- Keep updated_at current on per-user notification statistics
CREATE TRIGGER trigger_user_stats_updated_at
    BEFORE UPDATE ON notifications.user_stats
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();
