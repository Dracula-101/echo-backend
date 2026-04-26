-- =====================================================
-- NOTIFICATION SCHEMA - Push & In-App Notifications
-- =====================================================

-- Create Schema
CREATE SCHEMA IF NOT EXISTS notifications;

-- Notifications
CREATE TABLE notifications.notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Notification details
    notification_type VARCHAR(100) NOT NULL, -- message, mention, reaction, call, friend_request, group_invite
    notification_category VARCHAR(50), -- social, system, marketing, security
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    summary TEXT, -- Short version for badges
    
    -- Content
    icon_url TEXT,
    image_url TEXT,
    sound VARCHAR(100) DEFAULT 'default',
    badge_count INTEGER,
    
    -- Related entities
    related_user_id UUID REFERENCES auth.users(id),
    related_message_id UUID REFERENCES messages.messages(id),
    related_conversation_id UUID REFERENCES messages.conversations(id),
    related_call_id UUID REFERENCES messages.calls(id),
    
    -- Action data
    action_url TEXT,
    action_type VARCHAR(100), -- open_chat, open_profile, open_call, none
    action_data JSONB DEFAULT '{}'::JSONB,
    
    -- Status
    is_read BOOLEAN DEFAULT FALSE,
    is_seen BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    seen_at TIMESTAMPTZ,
    
    -- Delivery
    delivery_status VARCHAR(50) DEFAULT 'pending', -- pending, sent, delivered, failed
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_reason TEXT,
    retry_count INTEGER DEFAULT 0,
    
    -- Priority & Scheduling
    priority VARCHAR(20) DEFAULT 'normal', -- low, normal, high, urgent
    scheduled_for TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    
    -- Grouping (for notification bundling)
    group_key VARCHAR(255), -- For grouping similar notifications
    group_count INTEGER DEFAULT 1,
    is_group_summary BOOLEAN DEFAULT FALSE,
    
    -- Device targeting
    device_id VARCHAR(255),
    platform VARCHAR(50), -- ios, android, web
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Push Notification Delivery Log
CREATE TABLE notifications.push_delivery_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications.notifications(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    device_id VARCHAR(255),
    
    -- Token info
    push_token TEXT NOT NULL,
    push_provider VARCHAR(50) NOT NULL, -- fcm, apns
    
    -- Delivery details
    status VARCHAR(50) DEFAULT 'pending', -- pending, sent, delivered, failed, expired
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    dismissed_at TIMESTAMPTZ,
    
    -- Response from provider
    provider_message_id VARCHAR(255),
    provider_response JSONB,
    error_code VARCHAR(100),
    error_message TEXT,
    
    -- Metrics
    time_to_deliver_ms INTEGER,
    time_to_open_ms INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Email Notifications
CREATE TABLE notifications.email_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    notification_id UUID REFERENCES notifications.notifications(id) ON DELETE SET NULL,
    
    -- Email details
    email_to VARCHAR(255) NOT NULL,
    email_from VARCHAR(255) DEFAULT 'notifications@messaging.app',
    reply_to VARCHAR(255),
    subject VARCHAR(500) NOT NULL,
    body_text TEXT NOT NULL,
    body_html TEXT,
    
    -- Template
    template_name VARCHAR(100),
    template_data JSONB DEFAULT '{}'::JSONB,
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, sent, delivered, bounced, failed
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    clicked_at TIMESTAMPTZ,
    bounced_at TIMESTAMPTZ,
    bounce_reason TEXT,
    
    -- Provider details (GCP/SendGrid/etc)
    provider VARCHAR(50) DEFAULT 'gcp',
    provider_message_id VARCHAR(255),
    provider_response JSONB,
    
    -- Tracking
    open_count INTEGER DEFAULT 0,
    click_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- SMS Notifications
CREATE TABLE notifications.sms_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    notification_id UUID REFERENCES notifications.notifications(id) ON DELETE SET NULL,
    
    -- SMS details
    phone_number VARCHAR(20) NOT NULL,
    country_code VARCHAR(5),
    message TEXT NOT NULL,
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, sent, delivered, failed, undelivered
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    failure_reason TEXT,
    
    -- Provider details (GCP/Twilio/etc)
    provider VARCHAR(50) DEFAULT 'gcp',
    provider_message_id VARCHAR(255),
    provider_response JSONB,
    
    -- Cost tracking
    segment_count INTEGER DEFAULT 1,
    cost_per_sms DECIMAL(10,4),
    total_cost DECIMAL(10,4),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Notification Preferences (per user)
CREATE TABLE notifications.user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Global settings
    push_enabled BOOLEAN DEFAULT TRUE,
    email_enabled BOOLEAN DEFAULT TRUE,
    sms_enabled BOOLEAN DEFAULT FALSE,
    in_app_enabled BOOLEAN DEFAULT TRUE,
    
    -- Message notifications
    message_push BOOLEAN DEFAULT TRUE,
    message_email BOOLEAN DEFAULT FALSE,
    message_sms BOOLEAN DEFAULT FALSE,
    
    -- Mention notifications
    mention_push BOOLEAN DEFAULT TRUE,
    mention_email BOOLEAN DEFAULT TRUE,
    mention_sms BOOLEAN DEFAULT FALSE,
    
    -- Reaction notifications
    reaction_push BOOLEAN DEFAULT TRUE,
    reaction_email BOOLEAN DEFAULT FALSE,
    reaction_sms BOOLEAN DEFAULT FALSE,
    
    -- Call notifications
    call_push BOOLEAN DEFAULT TRUE,
    call_email BOOLEAN DEFAULT FALSE,
    call_sms BOOLEAN DEFAULT TRUE,
    missed_call_push BOOLEAN DEFAULT TRUE,
    
    -- Social notifications
    friend_request_push BOOLEAN DEFAULT TRUE,
    friend_request_email BOOLEAN DEFAULT TRUE,
    friend_accept_push BOOLEAN DEFAULT TRUE,
    
    -- Group notifications
    group_invite_push BOOLEAN DEFAULT TRUE,
    group_invite_email BOOLEAN DEFAULT TRUE,
    group_message_push BOOLEAN DEFAULT TRUE,
    group_mention_push BOOLEAN DEFAULT TRUE,
    
    -- System notifications
    security_alerts_push BOOLEAN DEFAULT TRUE,
    security_alerts_email BOOLEAN DEFAULT TRUE,
    security_alerts_sms BOOLEAN DEFAULT TRUE,
    account_updates_email BOOLEAN DEFAULT TRUE,
    
    -- Marketing
    marketing_push BOOLEAN DEFAULT FALSE,
    marketing_email BOOLEAN DEFAULT FALSE,
    promotional_email BOOLEAN DEFAULT FALSE,
    
    -- Quiet hours
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    quiet_hours_timezone VARCHAR(100),
    quiet_hours_days INTEGER[], -- 0=Sunday, 6=Saturday
    
    -- Notification bundling
    bundle_notifications BOOLEAN DEFAULT TRUE,
    bundle_interval_minutes INTEGER DEFAULT 5,
    
    -- Sound & Vibration
    notification_sound VARCHAR(100) DEFAULT 'default',
    vibration_enabled BOOLEAN DEFAULT TRUE,
    led_notification BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification Channels (for conversation-specific settings)
CREATE TABLE notifications.conversation_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    
    -- Override settings
    notifications_enabled BOOLEAN DEFAULT TRUE,
    push_enabled BOOLEAN,
    email_enabled BOOLEAN,
    sound_enabled BOOLEAN,
    vibration_enabled BOOLEAN,
    
    -- Custom sound
    custom_sound VARCHAR(100),
    
    -- Mute settings
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    mute_reason VARCHAR(100), -- manual, auto_mute, spam
    
    -- Priority
    priority_level VARCHAR(20) DEFAULT 'normal', -- low, normal, high
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, conversation_id)
);

-- Notification Templates
CREATE TABLE notifications.templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name VARCHAR(100) UNIQUE NOT NULL,
    template_type VARCHAR(50) NOT NULL, -- push, email, sms, in_app
    
    -- Content
    title_template TEXT,
    body_template TEXT NOT NULL,
    html_template TEXT, -- For emails
    
    -- Variables
    required_variables TEXT[],
    optional_variables TEXT[],
    
    -- Localization
    language_code VARCHAR(10) DEFAULT 'en',
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    version INTEGER DEFAULT 1,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by_user_id UUID REFERENCES auth.users(id)
);

-- Notification Actions (buttons in notifications)
CREATE TABLE notifications.notification_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications.notifications(id) ON DELETE CASCADE,
    action_id VARCHAR(100) NOT NULL, -- reply, accept, decline, view, etc
    action_label VARCHAR(100) NOT NULL,
    action_type VARCHAR(50), -- button, reply_input, dismiss
    action_url TEXT,
    action_data JSONB DEFAULT '{}'::JSONB,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification Action Responses
CREATE TABLE notifications.action_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications.notifications(id) ON DELETE CASCADE,
    action_id VARCHAR(100) NOT NULL,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    response_type VARCHAR(50), -- clicked, replied, dismissed
    response_data JSONB DEFAULT '{}'::JSONB,
    device_id VARCHAR(255),
    responded_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification Batches (for bulk sending)
CREATE TABLE notifications.batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_name VARCHAR(255),
    batch_type VARCHAR(50), -- broadcast, segment, individual
    
    -- Target
    target_user_ids UUID[],
    target_segment VARCHAR(100), -- all_users, premium_users, active_users, etc
    target_count INTEGER,
    
    -- Content
    notification_data JSONB NOT NULL,
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed, cancelled
    priority VARCHAR(20) DEFAULT 'normal',
    
    -- Progress
    sent_count INTEGER DEFAULT 0,
    delivered_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    opened_count INTEGER DEFAULT 0,
    
    -- Scheduling
    scheduled_for TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Limits
    rate_limit_per_second INTEGER DEFAULT 100,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by_user_id UUID REFERENCES auth.users(id),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- In-App Announcements
CREATE TABLE notifications.announcements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Content
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    announcement_type VARCHAR(50), -- feature, maintenance, promotion, warning
    severity VARCHAR(20) DEFAULT 'info', -- info, warning, error, success
    
    -- Display
    icon_url TEXT,
    image_url TEXT,
    background_color VARCHAR(7),
    
    -- Action
    action_label VARCHAR(100),
    action_url TEXT,
    action_type VARCHAR(50),
    
    -- Targeting
    target_audience VARCHAR(100) DEFAULT 'all', -- all, premium, free, ios, android, web
    target_user_ids UUID[],
    min_app_version VARCHAR(50),
    max_app_version VARCHAR(50),
    target_countries VARCHAR(5)[],
    
    -- Display rules
    display_frequency VARCHAR(50) DEFAULT 'once', -- once, daily, session, always
    max_display_count INTEGER DEFAULT 1,
    display_priority INTEGER DEFAULT 5,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    is_dismissible BOOLEAN DEFAULT TRUE,
    
    -- Schedule
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    
    -- Stats
    view_count INTEGER DEFAULT 0,
    click_count INTEGER DEFAULT 0,
    dismiss_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by_user_id UUID REFERENCES auth.users(id)
);

-- Announcement Views
CREATE TABLE notifications.announcement_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    announcement_id UUID NOT NULL REFERENCES notifications.announcements(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    view_count INTEGER DEFAULT 1,
    first_viewed_at TIMESTAMPTZ DEFAULT NOW(),
    last_viewed_at TIMESTAMPTZ DEFAULT NOW(),
    clicked BOOLEAN DEFAULT FALSE,
    clicked_at TIMESTAMPTZ,
    dismissed BOOLEAN DEFAULT FALSE,
    dismissed_at TIMESTAMPTZ,
    UNIQUE(announcement_id, user_id)
);

-- Notification Statistics (per user)
CREATE TABLE notifications.user_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    total_notifications_sent INTEGER DEFAULT 0,
    total_notifications_delivered INTEGER DEFAULT 0,
    total_notifications_opened INTEGER DEFAULT 0,
    total_notifications_dismissed INTEGER DEFAULT 0,
    
    push_sent INTEGER DEFAULT 0,
    push_delivered INTEGER DEFAULT 0,
    push_opened INTEGER DEFAULT 0,
    
    email_sent INTEGER DEFAULT 0,
    email_delivered INTEGER DEFAULT 0,
    email_opened INTEGER DEFAULT 0,
    email_clicked INTEGER DEFAULT 0,
    
    sms_sent INTEGER DEFAULT 0,
    sms_delivered INTEGER DEFAULT 0,
    
    last_notification_at TIMESTAMPTZ,
    last_opened_notification_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification Subscriptions (for topics/channels)
CREATE TABLE notifications.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    topic_name VARCHAR(255) NOT NULL, -- news, updates, promotions, etc
    is_subscribed BOOLEAN DEFAULT TRUE,
    subscribed_at TIMESTAMPTZ DEFAULT NOW(),
    unsubscribed_at TIMESTAMPTZ,
    UNIQUE(user_id, topic_name)
);