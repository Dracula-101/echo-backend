-- =====================================================
-- NOTIFICATIONS SCHEMA - ROW LEVEL SECURITY (RLS)
-- =====================================================

-- Enable RLS on all notifications tables
ALTER TABLE notifications.notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.push_delivery_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.email_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.sms_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.user_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.conversation_channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.notification_actions ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.action_responses ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.announcements ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.announcement_views ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.user_stats ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.subscriptions ENABLE ROW LEVEL SECURITY;

-- =====================================================
-- NOTIFICATIONS TABLE POLICIES
-- =====================================================

-- Users can view their own notifications
CREATE POLICY notifications_select_own
    ON notifications.notifications
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        AND deleted_at IS NULL
    );

-- Service role can create notifications
CREATE POLICY notifications_insert_service
    ON notifications.notifications
    FOR INSERT
    WITH CHECK (TRUE);

-- Users can update their own notifications (mark as read, etc.)
CREATE POLICY notifications_update_own
    ON notifications.notifications
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (
        user_id = auth.current_user_id()
    );

-- Users can delete their own notifications
CREATE POLICY notifications_delete_own
    ON notifications.notifications
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Admins can manage all notifications
CREATE POLICY notifications_admin_all
    ON notifications.notifications
    FOR ALL
    USING (auth.is_admin());

-- =====================================================
-- PUSH DELIVERY LOG POLICIES
-- =====================================================

-- Users can view their own push delivery logs
CREATE POLICY push_delivery_select_own
    ON notifications.push_delivery_log
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Service role can manage push delivery logs
CREATE POLICY push_delivery_service_all
    ON notifications.push_delivery_log
    FOR ALL
    USING (TRUE);

-- =====================================================
-- EMAIL NOTIFICATIONS POLICIES
-- =====================================================

-- Users can view their own email notifications
CREATE POLICY email_notifications_select_own
    ON notifications.email_notifications
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Service role can manage email notifications
CREATE POLICY email_notifications_service_all
    ON notifications.email_notifications
    FOR ALL
    USING (TRUE);

-- =====================================================
-- SMS NOTIFICATIONS POLICIES
-- =====================================================

-- Users can view their own SMS notifications
CREATE POLICY sms_notifications_select_own
    ON notifications.sms_notifications
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Service role can manage SMS notifications
CREATE POLICY sms_notifications_service_all
    ON notifications.sms_notifications
    FOR ALL
    USING (TRUE);

-- =====================================================
-- USER PREFERENCES POLICIES
-- =====================================================

-- Users can manage their own notification preferences
CREATE POLICY user_preferences_all_own
    ON notifications.user_preferences
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- CONVERSATION CHANNELS POLICIES
-- =====================================================

-- Users can manage their own conversation notification settings
CREATE POLICY conversation_channels_all_own
    ON notifications.conversation_channels
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- TEMPLATES POLICIES
-- =====================================================

-- Everyone can view active templates (for preview purposes)
CREATE POLICY templates_select_active
    ON notifications.templates
    FOR SELECT
    USING (is_active = TRUE);

-- Admins can manage templates
CREATE POLICY templates_admin_all
    ON notifications.templates
    FOR ALL
    USING (auth.is_admin());

-- =====================================================
-- NOTIFICATION ACTIONS POLICIES
-- =====================================================

-- Users can view actions for their notifications
CREATE POLICY notification_actions_select
    ON notifications.notification_actions
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM notifications.notifications n
            WHERE n.id = notifications.notification_actions.notification_id
            AND n.user_id = auth.current_user_id()
        )
    );

-- Service role can manage notification actions
CREATE POLICY notification_actions_service_all
    ON notifications.notification_actions
    FOR ALL
    USING (TRUE);

-- =====================================================
-- ACTION RESPONSES POLICIES
-- =====================================================

-- Users can view their own action responses
CREATE POLICY action_responses_select_own
    ON notifications.action_responses
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can create responses to their notifications
CREATE POLICY action_responses_insert_own
    ON notifications.action_responses
    FOR INSERT
    WITH CHECK (
        user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM notifications.notifications n
            WHERE n.id = notifications.action_responses.notification_id
            AND n.user_id = auth.current_user_id()
        )
    );

-- =====================================================
-- BATCHES POLICIES
-- =====================================================

-- Admins can manage notification batches
CREATE POLICY batches_admin_all
    ON notifications.batches
    FOR ALL
    USING (auth.is_admin());

-- Users can view batches they created
CREATE POLICY batches_select_creator
    ON notifications.batches
    FOR SELECT
    USING (created_by_user_id = auth.current_user_id());

-- =====================================================
-- ANNOUNCEMENTS POLICIES
-- =====================================================

-- Everyone can view active announcements targeted to them
CREATE POLICY announcements_select_active
    ON notifications.announcements
    FOR SELECT
    USING (
        is_active = TRUE
        AND (starts_at IS NULL OR starts_at <= NOW())
        AND (ends_at IS NULL OR ends_at > NOW())
        AND (
            target_audience = 'all'
            OR auth.current_user_id() = ANY(target_user_ids)
        )
    );

-- Admins can manage announcements
CREATE POLICY announcements_admin_all
    ON notifications.announcements
    FOR ALL
    USING (auth.is_admin());

-- =====================================================
-- ANNOUNCEMENT VIEWS POLICIES
-- =====================================================

-- Users can view their own announcement views
CREATE POLICY announcement_views_select_own
    ON notifications.announcement_views
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can create views for announcements they can see
CREATE POLICY announcement_views_insert_own
    ON notifications.announcement_views
    FOR INSERT
    WITH CHECK (
        user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM notifications.announcements a
            WHERE a.id = notifications.announcement_views.announcement_id
            AND a.is_active = TRUE
            AND (a.starts_at IS NULL OR a.starts_at <= NOW())
            AND (a.ends_at IS NULL OR a.ends_at > NOW())
        )
    );

-- Users can update their own views
CREATE POLICY announcement_views_update_own
    ON notifications.announcement_views
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- USER STATS POLICIES
-- =====================================================

-- Users can view their own notification stats
CREATE POLICY user_stats_select_own
    ON notifications.user_stats
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Service role can manage notification stats
CREATE POLICY user_stats_service_all
    ON notifications.user_stats
    FOR ALL
    USING (TRUE);

-- Admins can view all stats
CREATE POLICY user_stats_select_admin
    ON notifications.user_stats
    FOR SELECT
    USING (auth.is_admin());

-- =====================================================
-- SUBSCRIPTIONS POLICIES
-- =====================================================

-- Users can manage their own subscriptions
CREATE POLICY subscriptions_all_own
    ON notifications.subscriptions
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());