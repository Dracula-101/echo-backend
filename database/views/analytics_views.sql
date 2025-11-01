-- =====================================================
-- ANALYTICS VIEWS
-- =====================================================

-- Daily active users view
CREATE OR REPLACE VIEW analytics.daily_active_users AS
SELECT 
    DATE(created_at) as date,
    COUNT(DISTINCT user_id) as active_users
FROM analytics.events
WHERE event_type IN ('login', 'message_sent', 'profile_viewed')
AND created_at >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- Monthly active users view
CREATE OR REPLACE VIEW analytics.monthly_active_users AS
SELECT 
    DATE_TRUNC('month', created_at) as month,
    COUNT(DISTINCT user_id) as active_users
FROM analytics.events
WHERE event_type IN ('login', 'message_sent', 'profile_viewed')
AND created_at >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY DATE_TRUNC('month', created_at)
ORDER BY month DESC;

-- Message statistics view
CREATE OR REPLACE VIEW analytics.message_statistics AS
SELECT 
    DATE(m.created_at) as date,
    COUNT(*) as total_messages,
    COUNT(DISTINCT m.sender_id) as unique_senders,
    COUNT(DISTINCT m.conversation_id) as active_conversations,
    AVG(LENGTH(m.content)) as avg_message_length,
    COUNT(CASE WHEN m.has_attachments THEN 1 END) as messages_with_attachments
FROM messages.messages m
WHERE m.created_at >= CURRENT_DATE - INTERVAL '90 days'
AND m.deleted_at IS NULL
GROUP BY DATE(m.created_at)
ORDER BY date DESC;

-- User growth view
CREATE OR REPLACE VIEW analytics.user_growth AS
SELECT 
    DATE(created_at) as date,
    COUNT(*) as new_users,
    SUM(COUNT(*)) OVER (ORDER BY DATE(created_at)) as cumulative_users
FROM auth.users
WHERE created_at >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- Conversation activity view
CREATE OR REPLACE VIEW analytics.conversation_activity AS
SELECT 
    c.id as conversation_id,
    c.conversation_type,
    c.member_count,
    c.message_count,
    c.last_message_at,
    COUNT(DISTINCT m.sender_id) as unique_senders,
    EXTRACT(EPOCH FROM (MAX(m.created_at) - MIN(m.created_at))) / 3600 as conversation_duration_hours
FROM messages.conversations c
LEFT JOIN messages.messages m ON m.conversation_id = c.id
WHERE c.created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY c.id, c.conversation_type, c.member_count, c.message_count, c.last_message_at
ORDER BY c.message_count DESC;

-- User engagement metrics view
CREATE OR REPLACE VIEW analytics.user_engagement AS
SELECT 
    u.id as user_id,
    p.username,
    COUNT(DISTINCT s.id) as total_sessions,
    MAX(s.last_activity_at) as last_active_at,
    COUNT(DISTINCT m.id) as messages_sent,
    COUNT(DISTINCT m.conversation_id) as conversations_participated,
    COUNT(DISTINCT mf.id) as media_uploaded
FROM auth.users u
JOIN users.profiles p ON p.user_id = u.id
LEFT JOIN auth.sessions s ON s.user_id = u.id AND s.created_at >= CURRENT_DATE - INTERVAL '30 days'
LEFT JOIN messages.messages m ON m.sender_id = u.id AND m.created_at >= CURRENT_DATE - INTERVAL '30 days'
LEFT JOIN media.files mf ON mf.uploaded_by_user_id = u.id AND mf.created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY u.id, p.username
ORDER BY messages_sent DESC;

-- Peak usage hours view
CREATE OR REPLACE VIEW analytics.peak_usage_hours AS
SELECT 
    EXTRACT(HOUR FROM created_at) as hour_of_day,
    COUNT(*) as event_count,
    COUNT(DISTINCT user_id) as unique_users
FROM analytics.events
WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY EXTRACT(HOUR FROM created_at)
ORDER BY hour_of_day;

-- Device analytics view
CREATE OR REPLACE VIEW analytics.device_analytics AS
SELECT 
    s.device_type,
    s.device_os,
    COUNT(DISTINCT s.user_id) as unique_users,
    COUNT(*) as total_sessions,
    AVG(EXTRACT(EPOCH FROM (s.last_activity_at - s.created_at))) / 60 as avg_session_duration_minutes
FROM auth.sessions s
WHERE s.created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY s.device_type, s.device_os
ORDER BY unique_users DESC;

-- Media usage statistics view
CREATE OR REPLACE VIEW analytics.media_statistics AS
SELECT 
    DATE(mf.created_at) as date,
    mf.file_type,
    COUNT(*) as files_uploaded,
    SUM(mf.file_size) / 1024 / 1024 as total_size_mb,
    AVG(mf.file_size) / 1024 as avg_file_size_kb,
    COUNT(DISTINCT mf.uploaded_by_user_id) as unique_uploaders
FROM media.files mf
WHERE mf.created_at >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE(mf.created_at), mf.file_type
ORDER BY date DESC, total_size_mb DESC;

-- Notification delivery metrics view
CREATE OR REPLACE VIEW analytics.notification_metrics AS
SELECT 
    DATE(n.created_at) as date,
    n.notification_type,
    COUNT(*) as total_sent,
    COUNT(CASE WHEN n.is_read THEN 1 END) as total_read,
    COUNT(CASE WHEN n.read_at IS NOT NULL THEN 1 END) as total_opened,
    ROUND(COUNT(CASE WHEN n.is_read THEN 1 END)::NUMERIC / NULLIF(COUNT(*), 0) * 100, 2) as read_rate_percent,
    AVG(EXTRACT(EPOCH FROM (n.read_at - n.created_at)) / 60) as avg_time_to_read_minutes
FROM notifications.notifications n
WHERE n.created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY DATE(n.created_at), n.notification_type
ORDER BY date DESC, total_sent DESC;

-- User retention view
CREATE OR REPLACE VIEW analytics.user_retention AS
WITH cohorts AS (
    SELECT 
        user_id,
        DATE_TRUNC('month', created_at) as cohort_month
    FROM auth.users
),
user_activity AS (
    SELECT 
        user_id,
        DATE_TRUNC('month', created_at) as activity_month
    FROM analytics.events
    WHERE event_type IN ('login', 'message_sent')
)
SELECT 
    c.cohort_month,
    COUNT(DISTINCT c.user_id) as cohort_size,
    ua.activity_month,
    COUNT(DISTINCT ua.user_id) as active_users,
    ROUND(COUNT(DISTINCT ua.user_id)::NUMERIC / COUNT(DISTINCT c.user_id) * 100, 2) as retention_percent
FROM cohorts c
LEFT JOIN user_activity ua ON c.user_id = ua.user_id
WHERE c.cohort_month >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY c.cohort_month, ua.activity_month
ORDER BY c.cohort_month DESC, ua.activity_month;

-- Top conversations by activity
CREATE OR REPLACE VIEW analytics.top_conversations AS
SELECT 
    c.id,
    c.conversation_type,
    c.title,
    c.member_count,
    c.message_count,
    COUNT(DISTINCT m.sender_id) as unique_participants,
    MAX(m.created_at) as last_message_at,
    EXTRACT(EPOCH FROM (MAX(m.created_at) - MIN(m.created_at))) / 86400 as age_days
FROM messages.conversations c
LEFT JOIN messages.messages m ON m.conversation_id = c.id
WHERE c.created_at >= CURRENT_DATE - INTERVAL '30 days'
AND c.is_active = TRUE
GROUP BY c.id, c.conversation_type, c.title, c.member_count, c.message_count
ORDER BY c.message_count DESC
LIMIT 100;
