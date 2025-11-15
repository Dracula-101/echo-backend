-- =====================================================
-- NOTIFICATIONS SCHEMA - INDEXES (FIXED)
-- =====================================================

-- Notifications table indexes
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications.notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications.notifications(notification_type);
CREATE INDEX IF NOT EXISTS idx_notifications_category ON notifications.notifications(notification_category);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications.notifications(is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_seen ON notifications.notifications(is_seen);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications.notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_unseen ON notifications.notifications(user_id, is_seen) WHERE is_seen = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications.notifications(delivery_status);
CREATE INDEX IF NOT EXISTS idx_notifications_priority ON notifications.notifications(priority);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications.notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_scheduled ON notifications.notifications(scheduled_for) 
    WHERE scheduled_for IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_expires ON notifications.notifications(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_group ON notifications.notifications(group_key) WHERE group_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_deleted ON notifications.notifications(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_related_user ON notifications.notifications(related_user_id) WHERE related_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_related_message ON notifications.notifications(related_message_id) WHERE related_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_related_conversation ON notifications.notifications(related_conversation_id) WHERE related_conversation_id IS NOT NULL;

-- Push delivery log table indexes
CREATE INDEX IF NOT EXISTS idx_push_delivery_notification ON notifications.push_delivery_log(notification_id);
CREATE INDEX IF NOT EXISTS idx_push_delivery_user ON notifications.push_delivery_log(user_id);
CREATE INDEX IF NOT EXISTS idx_push_delivery_device ON notifications.push_delivery_log(device_id);
CREATE INDEX IF NOT EXISTS idx_push_delivery_status ON notifications.push_delivery_log(status);
CREATE INDEX IF NOT EXISTS idx_push_delivery_created ON notifications.push_delivery_log(created_at);
CREATE INDEX IF NOT EXISTS idx_push_delivery_opened ON notifications.push_delivery_log(opened_at) WHERE opened_at IS NOT NULL;

-- Email notifications table indexes
CREATE INDEX IF NOT EXISTS idx_email_notifications_user ON notifications.email_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_email_notifications_notification ON notifications.email_notifications(notification_id);
CREATE INDEX IF NOT EXISTS idx_email_notifications_status ON notifications.email_notifications(status);
CREATE INDEX IF NOT EXISTS idx_email_notifications_created ON notifications.email_notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_email_notifications_sent ON notifications.email_notifications(sent_at);
CREATE INDEX IF NOT EXISTS idx_email_notifications_opened ON notifications.email_notifications(opened_at) WHERE opened_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_email_notifications_bounced ON notifications.email_notifications(bounced_at) WHERE bounced_at IS NOT NULL;

-- SMS notifications table indexes
CREATE INDEX IF NOT EXISTS idx_sms_notifications_user ON notifications.sms_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_sms_notifications_notification ON notifications.sms_notifications(notification_id);
CREATE INDEX IF NOT EXISTS idx_sms_notifications_phone ON notifications.sms_notifications(phone_number);
CREATE INDEX IF NOT EXISTS idx_sms_notifications_status ON notifications.sms_notifications(status);
CREATE INDEX IF NOT EXISTS idx_sms_notifications_created ON notifications.sms_notifications(created_at);

-- User preferences table indexes
CREATE INDEX IF NOT EXISTS idx_user_preferences_user ON notifications.user_preferences(user_id);

-- Conversation channels table indexes
CREATE INDEX IF NOT EXISTS idx_conversation_channels_user ON notifications.conversation_channels(user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_channels_conversation ON notifications.conversation_channels(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conversation_channels_muted ON notifications.conversation_channels(is_muted) WHERE is_muted = TRUE;

-- Templates table indexes
CREATE INDEX IF NOT EXISTS idx_templates_name ON notifications.templates(template_name);
CREATE INDEX IF NOT EXISTS idx_templates_type ON notifications.templates(template_type);
CREATE INDEX IF NOT EXISTS idx_templates_language ON notifications.templates(language_code);
CREATE INDEX IF NOT EXISTS idx_templates_active ON notifications.templates(is_active) WHERE is_active = TRUE;

-- Notification actions table indexes
CREATE INDEX IF NOT EXISTS idx_notification_actions_notification ON notifications.notification_actions(notification_id);
CREATE INDEX IF NOT EXISTS idx_notification_actions_id ON notifications.notification_actions(action_id);
CREATE INDEX IF NOT EXISTS idx_notification_actions_order ON notifications.notification_actions(notification_id, display_order);

-- Action responses table indexes
CREATE INDEX IF NOT EXISTS idx_action_responses_notification ON notifications.action_responses(notification_id);
CREATE INDEX IF NOT EXISTS idx_action_responses_user ON notifications.action_responses(user_id);
CREATE INDEX IF NOT EXISTS idx_action_responses_action ON notifications.action_responses(action_id);
CREATE INDEX IF NOT EXISTS idx_action_responses_responded ON notifications.action_responses(responded_at);

-- Batches table indexes
CREATE INDEX IF NOT EXISTS idx_batches_status ON notifications.batches(status);
CREATE INDEX IF NOT EXISTS idx_batches_priority ON notifications.batches(priority);
CREATE INDEX IF NOT EXISTS idx_batches_scheduled ON notifications.batches(scheduled_for) WHERE scheduled_for IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_batches_created ON notifications.batches(created_at);
CREATE INDEX IF NOT EXISTS idx_batches_created_by ON notifications.batches(created_by_user_id);

-- Announcements table indexes
CREATE INDEX IF NOT EXISTS idx_announcements_active ON notifications.announcements(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_announcements_type ON notifications.announcements(announcement_type);
CREATE INDEX IF NOT EXISTS idx_announcements_severity ON notifications.announcements(severity);
CREATE INDEX IF NOT EXISTS idx_announcements_starts ON notifications.announcements(starts_at);
CREATE INDEX IF NOT EXISTS idx_announcements_ends ON notifications.announcements(ends_at);
CREATE INDEX IF NOT EXISTS idx_announcements_audience ON notifications.announcements(target_audience);
CREATE INDEX IF NOT EXISTS idx_announcements_priority ON notifications.announcements(display_priority);
-- FIXED: Removed NOW() from index predicate
CREATE INDEX IF NOT EXISTS idx_announcements_active_period ON notifications.announcements(is_active, starts_at, ends_at) 
    WHERE is_active = TRUE;

-- Announcement views table indexes
CREATE INDEX IF NOT EXISTS idx_announcement_views_announcement ON notifications.announcement_views(announcement_id);
CREATE INDEX IF NOT EXISTS idx_announcement_views_user ON notifications.announcement_views(user_id);
CREATE INDEX IF NOT EXISTS idx_announcement_views_first_viewed ON notifications.announcement_views(first_viewed_at);
CREATE INDEX IF NOT EXISTS idx_announcement_views_clicked ON notifications.announcement_views(clicked) WHERE clicked = TRUE;
CREATE INDEX IF NOT EXISTS idx_announcement_views_dismissed ON notifications.announcement_views(dismissed) WHERE dismissed = TRUE;

-- User stats table indexes
CREATE INDEX IF NOT EXISTS idx_user_stats_user ON notifications.user_stats(user_id);
CREATE INDEX IF NOT EXISTS idx_user_stats_last_notification ON notifications.user_stats(last_notification_at);

-- Subscriptions table indexes
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON notifications.subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_topic ON notifications.subscriptions(topic_name);
CREATE INDEX IF NOT EXISTS idx_subscriptions_subscribed ON notifications.subscriptions(is_subscribed) WHERE is_subscribed = TRUE;