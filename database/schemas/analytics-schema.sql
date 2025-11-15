-- =====================================================
-- ANALYTICS SCHEMA - User Behavior & Business Metrics
-- =====================================================

-- Create Schema
CREATE SCHEMA IF NOT EXISTS analytics;

-- User Events (detailed event tracking)
CREATE TABLE analytics.events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    
    -- Event details
    event_name VARCHAR(255) NOT NULL, -- app_open, message_sent, profile_view, etc
    event_category VARCHAR(100), -- engagement, conversion, retention, monetization
    event_action VARCHAR(100),
    event_label VARCHAR(255),
    event_value DECIMAL(10,2),
    
    -- Context
    screen_name VARCHAR(255),
    previous_screen VARCHAR(255),
    flow_name VARCHAR(100), -- onboarding, messaging, settings
    
    -- Device & Platform
    device_id VARCHAR(255),
    platform VARCHAR(50), -- ios, android, web
    os_name VARCHAR(100),
    os_version VARCHAR(50),
    app_version VARCHAR(50),
    app_build VARCHAR(50),
    
    -- Location
    ip_address INET,
    country VARCHAR(100),
    region VARCHAR(100),
    city VARCHAR(100),
    timezone VARCHAR(100),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    
    -- Network
    connection_type VARCHAR(50), -- wifi, cellular, ethernet
    carrier VARCHAR(100),
    
    -- Performance
    event_duration_ms INTEGER,
    load_time_ms INTEGER,
    ttfb_ms INTEGER, -- Time to first byte
    
    -- Custom properties
    properties JSONB DEFAULT '{}'::JSONB,
    
    -- Attribution
    utm_source VARCHAR(255),
    utm_medium VARCHAR(255),
    utm_campaign VARCHAR(255),
    utm_term VARCHAR(255),
    utm_content VARCHAR(255),
    referrer TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    event_timestamp TIMESTAMPTZ DEFAULT NOW()
);

-- User Sessions (aggregated session data)
CREATE TABLE analytics.user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE CASCADE,
    
    -- Session details
    session_start TIMESTAMPTZ NOT NULL,
    session_end TIMESTAMPTZ,
    session_duration_seconds INTEGER,
    
    -- Activity
    page_views INTEGER DEFAULT 0,
    screen_views INTEGER DEFAULT 0,
    event_count INTEGER DEFAULT 0,
    messages_sent INTEGER DEFAULT 0,
    messages_received INTEGER DEFAULT 0,
    
    -- Device & Platform
    device_id VARCHAR(255),
    device_type VARCHAR(50),
    platform VARCHAR(50),
    app_version VARCHAR(50),
    
    -- Location
    country VARCHAR(100),
    city VARCHAR(100),
    ip_address INET,
    
    -- Engagement
    is_engaged BOOLEAN DEFAULT FALSE, -- >10s or 2+ screens
    bounce BOOLEAN DEFAULT TRUE,
    
    -- Attribution
    traffic_source VARCHAR(100),
    campaign VARCHAR(255),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Daily Active Users (DAU)
CREATE TABLE analytics.daily_active_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Activity metrics
    sessions_count INTEGER DEFAULT 1,
    messages_sent INTEGER DEFAULT 0,
    messages_received INTEGER DEFAULT 0,
    time_spent_seconds INTEGER DEFAULT 0,
    features_used TEXT[],
    
    -- Platform
    platforms_used VARCHAR(50)[], -- [ios, web]
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, user_id)
);

-- User Retention Cohorts
CREATE TABLE analytics.user_cohorts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    cohort_date DATE NOT NULL, -- Registration date
    cohort_week INTEGER, -- Week number since registration
    cohort_month INTEGER, -- Month number since registration
    
    -- Retention tracking
    day_1_active BOOLEAN DEFAULT FALSE,
    day_7_active BOOLEAN DEFAULT FALSE,
    day_14_active BOOLEAN DEFAULT FALSE,
    day_30_active BOOLEAN DEFAULT FALSE,
    day_60_active BOOLEAN DEFAULT FALSE,
    day_90_active BOOLEAN DEFAULT FALSE,
    
    -- Engagement
    messages_sent_total INTEGER DEFAULT 0,
    days_active_count INTEGER DEFAULT 0,
    last_active_date DATE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);

-- Funnel Analytics (conversion tracking)
CREATE TABLE analytics.funnels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    
    funnel_name VARCHAR(100) NOT NULL, -- onboarding, messaging, premium_upgrade
    step_name VARCHAR(100) NOT NULL,
    step_order INTEGER NOT NULL,
    
    -- Timing
    entered_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    time_in_step_seconds INTEGER,
    
    -- Status
    is_completed BOOLEAN DEFAULT FALSE,
    dropped_off BOOLEAN DEFAULT FALSE,
    drop_off_reason TEXT,
    
    -- Context
    properties JSONB DEFAULT '{}'::JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Feature Usage
CREATE TABLE analytics.feature_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    feature_name VARCHAR(100) NOT NULL, -- voice_call, video_call, stickers, etc
    feature_category VARCHAR(100),
    
    -- Usage stats
    usage_count INTEGER DEFAULT 1,
    first_used_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ DEFAULT NOW(),
    total_duration_seconds INTEGER DEFAULT 0,
    
    -- Value
    engagement_score DECIMAL(10,2), -- 0-100
    
    date DATE DEFAULT CURRENT_DATE,
    UNIQUE(user_id, feature_name, date)
);

-- A/B Test Variants
CREATE TABLE analytics.ab_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_name VARCHAR(255) UNIQUE NOT NULL,
    test_description TEXT,
    
    -- Configuration
    variants JSONB NOT NULL, -- [{name, weight, config}]
    
    -- Targeting
    target_percentage INTEGER DEFAULT 100, -- Percentage of users to include
    target_platforms VARCHAR(50)[],
    target_countries VARCHAR(5)[],
    target_user_segments TEXT[],
    
    -- Status
    status VARCHAR(50) DEFAULT 'draft', -- draft, active, paused, completed
    
    -- Schedule
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    
    -- Results
    sample_size INTEGER DEFAULT 0,
    confidence_level DECIMAL(5,2),
    winning_variant VARCHAR(100),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by_user_id UUID REFERENCES auth.users(id)
);

-- A/B Test Assignments
CREATE TABLE analytics.ab_test_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_id UUID NOT NULL REFERENCES analytics.ab_tests(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    variant_name VARCHAR(100) NOT NULL,
    variant_config JSONB,
    
    -- Metrics
    converted BOOLEAN DEFAULT FALSE,
    conversion_value DECIMAL(10,2),
    converted_at TIMESTAMPTZ,
    
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(test_id, user_id)
);

-- Performance Metrics
CREATE TABLE analytics.performance_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Metric details
    metric_type VARCHAR(100) NOT NULL, -- api_latency, db_query_time, page_load
    metric_name VARCHAR(255) NOT NULL,
    metric_value DECIMAL(15,4) NOT NULL,
    metric_unit VARCHAR(50), -- ms, seconds, bytes, count
    
    -- Context
    service_name VARCHAR(100), -- auth-service, message-service
    endpoint VARCHAR(255),
    method VARCHAR(20), -- GET, POST, etc
    status_code INTEGER,
    
    -- Performance
    duration_ms INTEGER,
    memory_used_mb INTEGER,
    cpu_percentage DECIMAL(5,2),
    
    -- Request details
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    ip_address INET,
    
    -- Error tracking
    is_error BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    error_stack TEXT,
    
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Error Logs
CREATE TABLE analytics.error_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Error details
    error_type VARCHAR(100) NOT NULL, -- exception, api_error, validation_error
    error_message TEXT NOT NULL,
    error_code VARCHAR(100),
    error_stack TEXT,
    
    -- Severity
    severity VARCHAR(20) DEFAULT 'error', -- debug, info, warning, error, critical
    
    -- Context
    service_name VARCHAR(100),
    function_name VARCHAR(255),
    file_path TEXT,
    line_number INTEGER,
    
    -- User context
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    device_id VARCHAR(255),
    
    -- Request context
    http_method VARCHAR(20),
    endpoint VARCHAR(255),
    request_id VARCHAR(255),
    request_body JSONB,
    response_body JSONB,
    
    -- Environment
    environment VARCHAR(50), -- development, staging, production
    app_version VARCHAR(50),
    platform VARCHAR(50),
    
    -- Frequency
    occurrences INTEGER DEFAULT 1,
    first_occurred_at TIMESTAMPTZ DEFAULT NOW(),
    last_occurred_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Resolution
    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    resolved_by_user_id UUID REFERENCES auth.users(id),
    resolution_notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Revenue & Monetization
CREATE TABLE analytics.revenue_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Transaction details
    transaction_type VARCHAR(100) NOT NULL, -- subscription, in_app_purchase, upgrade
    product_id VARCHAR(255),
    product_name VARCHAR(255),
    product_category VARCHAR(100),
    
    -- Amount
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'USD',
    amount_usd DECIMAL(10,2), -- Normalized to USD
    
    -- Payment
    payment_method VARCHAR(100), -- credit_card, paypal, apple_pay, google_pay
    payment_provider VARCHAR(100),
    transaction_id VARCHAR(255) UNIQUE,
    
    -- Status
    status VARCHAR(50) DEFAULT 'completed', -- pending, completed, refunded, failed
    refunded_at TIMESTAMPTZ,
    refund_amount DECIMAL(10,2),
    refund_reason TEXT,
    
    -- Attribution
    campaign_source VARCHAR(255),
    
    -- Subscription specific
    is_subscription BOOLEAN DEFAULT FALSE,
    subscription_period VARCHAR(50), -- monthly, yearly
    is_trial BOOLEAN DEFAULT FALSE,
    is_renewal BOOLEAN DEFAULT FALSE,
    
    transaction_date TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Lifetime Value (LTV)
CREATE TABLE analytics.user_ltv (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Revenue
    total_revenue DECIMAL(10,2) DEFAULT 0.00,
    total_transactions INTEGER DEFAULT 0,
    average_transaction_value DECIMAL(10,2) DEFAULT 0.00,
    
    -- Engagement
    days_active INTEGER DEFAULT 0,
    messages_sent_total INTEGER DEFAULT 0,
    messages_received_total INTEGER DEFAULT 0,
    
    -- Predicted LTV
    predicted_ltv_30d DECIMAL(10,2),
    predicted_ltv_90d DECIMAL(10,2),
    predicted_ltv_365d DECIMAL(10,2),
    
    -- Segments
    user_segment VARCHAR(100), -- whale, dolphin, minnow, inactive
    churn_risk_score DECIMAL(5,2), -- 0-100
    
    last_calculated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Aggregated Daily Metrics
CREATE TABLE analytics.daily_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL UNIQUE,
    
    -- User metrics
    dau INTEGER DEFAULT 0, -- Daily Active Users
    new_users INTEGER DEFAULT 0,
    churned_users INTEGER DEFAULT 0,
    
    -- Engagement
    total_messages_sent BIGINT DEFAULT 0,
    total_sessions BIGINT DEFAULT 0,
    avg_session_duration_seconds INTEGER DEFAULT 0,
    avg_messages_per_user DECIMAL(10,2) DEFAULT 0.00,
    
    -- Content
    images_uploaded INTEGER DEFAULT 0,
    videos_uploaded INTEGER DEFAULT 0,
    voice_messages_sent INTEGER DEFAULT 0,
    
    -- Social
    new_conversations INTEGER DEFAULT 0,
    new_groups_created INTEGER DEFAULT 0,
    new_friendships INTEGER DEFAULT 0,
    
    -- Calls
    voice_calls_total INTEGER DEFAULT 0,
    video_calls_total INTEGER DEFAULT 0,
    total_call_duration_minutes BIGINT DEFAULT 0,
    
    -- Revenue
    revenue_total DECIMAL(10,2) DEFAULT 0.00,
    new_subscriptions INTEGER DEFAULT 0,
    cancelled_subscriptions INTEGER DEFAULT 0,
    
    -- Performance
    avg_api_latency_ms INTEGER,
    error_count INTEGER DEFAULT 0,
    uptime_percentage DECIMAL(5,2) DEFAULT 100.00,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Segments
CREATE TABLE analytics.user_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    segment_name VARCHAR(100) NOT NULL, -- power_user, casual_user, at_risk, churned
    segment_category VARCHAR(100),
    
    -- Criteria met
    criteria_met JSONB,
    confidence_score DECIMAL(5,2), -- 0-100
    
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    
    UNIQUE(user_id, segment_name)
);

-- Page/Screen Views
CREATE TABLE analytics.page_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    
    -- Page details
    page_url TEXT,
    page_title VARCHAR(255),
    screen_name VARCHAR(255),
    screen_class VARCHAR(255),
    
    -- Referrer
    referrer_url TEXT,
    referrer_source VARCHAR(255),
    
    -- Timing
    view_duration_seconds INTEGER,
    time_to_interactive_ms INTEGER,
    
    -- Engagement
    scroll_depth_percentage INTEGER,
    clicks_count INTEGER DEFAULT 0,
    
    -- Device
    device_id VARCHAR(255),
    platform VARCHAR(50),
    
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Search Analytics
CREATE TABLE analytics.search_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    
    -- Query details
    search_query TEXT NOT NULL,
    search_type VARCHAR(50), -- user_search, message_search, gif_search
    search_category VARCHAR(100),
    
    -- Results
    results_count INTEGER DEFAULT 0,
    results_clicked INTEGER DEFAULT 0,
    clicked_position INTEGER, -- Position of clicked result
    
    -- Timing
    search_duration_ms INTEGER,
    
    -- Context
    screen_name VARCHAR(255),
    
    searched_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Content Engagement
CREATE TABLE analytics.content_engagement (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    content_type VARCHAR(100) NOT NULL, -- message, status, media, sticker
    content_id UUID NOT NULL,
    
    -- Engagement metrics
    viewed BOOLEAN DEFAULT FALSE,
    viewed_at TIMESTAMPTZ,
    view_duration_ms INTEGER,
    
    liked BOOLEAN DEFAULT FALSE,
    liked_at TIMESTAMPTZ,
    
    shared BOOLEAN DEFAULT FALSE,
    shared_at TIMESTAMPTZ,
    share_count INTEGER DEFAULT 0,
    
    saved BOOLEAN DEFAULT FALSE,
    saved_at TIMESTAMPTZ,
    
    commented BOOLEAN DEFAULT FALSE,
    comment_count INTEGER DEFAULT 0,
    
    -- Completion (for video/audio)
    completion_percentage INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, content_type, content_id)
);

-- Push Notification Analytics
CREATE TABLE analytics.push_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    
    notification_type VARCHAR(100),
    platform VARCHAR(50),
    
    -- Metrics
    sent_count INTEGER DEFAULT 0,
    delivered_count INTEGER DEFAULT 0,
    opened_count INTEGER DEFAULT 0,
    dismissed_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    
    -- Rates
    delivery_rate DECIMAL(5,2) DEFAULT 0.00,
    open_rate DECIMAL(5,2) DEFAULT 0.00,
    
    -- Timing
    avg_delivery_time_seconds INTEGER,
    avg_time_to_open_seconds INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, notification_type, platform)
);

-- API Usage Metrics
CREATE TABLE analytics.api_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- API details
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(20) NOT NULL,
    service_name VARCHAR(100),
    
    -- Authentication
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    api_key_id UUID REFERENCES auth.api_keys(id) ON DELETE SET NULL,
    
    -- Request
    request_size_bytes INTEGER,
    request_headers JSONB,
    query_params JSONB,
    
    -- Response
    status_code INTEGER NOT NULL,
    response_size_bytes INTEGER,
    response_time_ms INTEGER,
    
    -- Performance
    db_query_time_ms INTEGER,
    cache_hit BOOLEAN DEFAULT FALSE,
    
    -- Location
    ip_address INET,
    country VARCHAR(100),
    
    -- Error
    is_error BOOLEAN DEFAULT FALSE,
    error_code VARCHAR(100),
    error_message TEXT,
    
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Webhook Deliveries (for analytics on outgoing webhooks)
CREATE TABLE analytics.webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    webhook_url TEXT NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    
    -- Delivery
    status VARCHAR(50) DEFAULT 'pending', -- pending, sent, failed, retrying
    http_status_code INTEGER,
    response_time_ms INTEGER,
    
    -- Retry logic
    attempt_number INTEGER DEFAULT 1,
    max_attempts INTEGER DEFAULT 3,
    next_retry_at TIMESTAMPTZ,
    
    -- Payload
    payload_size_bytes INTEGER,
    
    -- Error
    error_message TEXT,
    
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Feedback & Ratings
CREATE TABLE analytics.user_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Feedback type
    feedback_type VARCHAR(100) NOT NULL, -- bug_report, feature_request, rating, complaint
    feedback_category VARCHAR(100),
    
    -- Content
    rating INTEGER, -- 1-5 stars
    title VARCHAR(255),
    description TEXT,
    
    -- Context
    screen_name VARCHAR(255),
    app_version VARCHAR(50),
    platform VARCHAR(50),
    
    -- Attachments
    screenshot_urls TEXT[],
    log_data JSONB,
    
    -- Status
    status VARCHAR(50) DEFAULT 'new', -- new, reviewed, in_progress, resolved, closed
    priority VARCHAR(20) DEFAULT 'medium',
    
    -- Response
    response_text TEXT,
    responded_at TIMESTAMPTZ,
    responded_by_user_id UUID REFERENCES auth.users(id),
    
    -- Sentiment
    sentiment_score DECIMAL(5,2), -- -100 to +100
    sentiment_label VARCHAR(50), -- negative, neutral, positive
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Referral Tracking
CREATE TABLE analytics.referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    referred_user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    
    -- Referral method
    referral_code VARCHAR(100) UNIQUE NOT NULL,
    referral_link TEXT,
    referral_method VARCHAR(50), -- link, qr_code, contact_invite, social_share
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, completed, expired
    
    -- Tracking
    clicked_at TIMESTAMPTZ,
    registered_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Rewards
    reward_given BOOLEAN DEFAULT FALSE,
    reward_type VARCHAR(100),
    reward_value DECIMAL(10,2),
    reward_given_at TIMESTAMPTZ,
    
    -- Attribution
    utm_source VARCHAR(255),
    utm_campaign VARCHAR(255),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Churn Prediction
CREATE TABLE analytics.churn_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Prediction
    churn_probability DECIMAL(5,2) NOT NULL, -- 0-100
    churn_risk_level VARCHAR(20), -- low, medium, high, critical
    
    -- Factors
    contributing_factors JSONB, -- [{factor, weight, value}]
    
    -- Timing
    predicted_churn_date DATE,
    days_until_churn INTEGER,
    
    -- Intervention
    intervention_recommended VARCHAR(255),
    intervention_sent BOOLEAN DEFAULT FALSE,
    intervention_sent_at TIMESTAMPTZ,
    
    -- Outcome
    actually_churned BOOLEAN,
    churn_date DATE,
    
    prediction_date DATE DEFAULT CURRENT_DATE,
    model_version VARCHAR(50),
    confidence_score DECIMAL(5,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Conversion Attribution
CREATE TABLE analytics.attribution (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Conversion
    conversion_type VARCHAR(100) NOT NULL, -- registration, first_message, subscription
    conversion_value DECIMAL(10,2),
    converted_at TIMESTAMPTZ NOT NULL,
    
    -- Attribution
    attribution_model VARCHAR(50) DEFAULT 'last_click', -- first_click, last_click, linear, time_decay
    
    -- Touchpoints
    first_touch_source VARCHAR(255),
    first_touch_medium VARCHAR(255),
    first_touch_campaign VARCHAR(255),
    first_touch_at TIMESTAMPTZ,
    
    last_touch_source VARCHAR(255),
    last_touch_medium VARCHAR(255),
    last_touch_campaign VARCHAR(255),
    last_touch_at TIMESTAMPTZ,
    
    -- Journey
    touchpoint_count INTEGER DEFAULT 0,
    journey_duration_hours INTEGER,
    touchpoints JSONB DEFAULT '[]'::JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Heatmaps (for UI interaction tracking)
CREATE TABLE analytics.heatmap_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    screen_name VARCHAR(255) NOT NULL,
    element_id VARCHAR(255),
    element_type VARCHAR(100), -- button, link, input, image
    
    -- Interaction
    interaction_type VARCHAR(50) NOT NULL, -- click, tap, hover, scroll
    
    -- Position
    x_coordinate INTEGER,
    y_coordinate INTEGER,
    viewport_width INTEGER,
    viewport_height INTEGER,
    
    -- Frequency
    interaction_count INTEGER DEFAULT 1,
    
    -- User context
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    
    -- Platform
    platform VARCHAR(50),
    device_type VARCHAR(50),
    
    date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);