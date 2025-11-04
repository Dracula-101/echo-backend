-- =====================================================
-- PRESENCE SCHEMA - Real-time User Presence Tracking
-- Description: Dedicated schema for presence-service
-- =====================================================

-- Create Schema
CREATE SCHEMA IF NOT EXISTS presence;

-- Real-time Presence Status
-- Tracks current online/offline status with sub-second accuracy
CREATE TABLE presence.user_status (
    user_id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'offline', -- online, offline, away, busy, invisible
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Connection details
    is_online BOOLEAN DEFAULT FALSE,
    active_connections INTEGER DEFAULT 0,

    -- Device tracking
    primary_device_id VARCHAR(255),
    devices JSONB DEFAULT '[]'::JSONB, -- [{device_id, device_type, connected_at}]

    -- Status message
    custom_status_text VARCHAR(255),
    custom_status_emoji VARCHAR(10),
    custom_status_expires_at TIMESTAMPTZ,

    -- Visibility settings
    show_last_seen BOOLEAN DEFAULT TRUE,
    show_online_status BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Active Connection Sessions
-- Tracks individual WebSocket/connection sessions per user
CREATE TABLE presence.connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    connection_id VARCHAR(255) UNIQUE NOT NULL,

    -- Connection details
    device_id VARCHAR(255),
    device_type VARCHAR(50), -- mobile, tablet, desktop, web
    device_os VARCHAR(100),
    browser_name VARCHAR(100),

    -- Network info
    ip_address INET,
    user_agent TEXT,

    -- Connection state
    status VARCHAR(20) DEFAULT 'connected', -- connected, idle, disconnected
    connected_at TIMESTAMPTZ DEFAULT NOW(),
    last_heartbeat_at TIMESTAMPTZ DEFAULT NOW(),
    disconnected_at TIMESTAMPTZ,

    -- Server routing
    server_id VARCHAR(255), -- For routing in distributed systems
    server_region VARCHAR(50),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Activity Log
-- Tracks user activities for presence analytics
CREATE TABLE presence.activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL, -- typing, viewing, recording, idle, active
    context_type VARCHAR(50), -- conversation, profile, settings
    context_id UUID, -- conversation_id, profile_id, etc.

    -- Activity details
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER,

    -- Device info
    device_id VARCHAR(255),
    connection_id VARCHAR(255),

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Typing Indicators
-- Ephemeral typing status (TTL: 5-10 seconds)
CREATE TABLE presence.typing_indicators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL, -- References messages.conversations
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,

    -- Typing state
    is_typing BOOLEAN DEFAULT TRUE,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL, -- Auto-expire after 10 seconds

    -- Typing details
    device_id VARCHAR(255),

    UNIQUE(conversation_id, user_id)
);

-- Presence Subscriptions
-- Tracks who is subscribed to whose presence updates
CREATE TABLE presence.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscriber_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    target_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,

    -- Subscription details
    subscription_type VARCHAR(50) DEFAULT 'contact', -- contact, conversation, temporary
    context_id UUID, -- conversation_id if applicable

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    subscribed_at TIMESTAMPTZ DEFAULT NOW(),
    unsubscribed_at TIMESTAMPTZ,
    last_notified_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(subscriber_user_id, target_user_id, subscription_type),
    CHECK (subscriber_user_id != target_user_id)
);

-- Presence History
-- For analytics and debugging
CREATE TABLE presence.status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    previous_status VARCHAR(20),
    new_status VARCHAR(20) NOT NULL,

    -- Session info
    connection_id VARCHAR(255),
    device_id VARCHAR(255),

    -- Timing
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    duration_seconds INTEGER, -- How long they were in previous_status

    -- Reason
    change_reason VARCHAR(50), -- login, logout, timeout, manual, system

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for Performance
CREATE INDEX idx_user_status_online ON presence.user_status(is_online, status);
CREATE INDEX idx_user_status_last_seen ON presence.user_status(last_seen_at);
CREATE INDEX idx_connections_user ON presence.connections(user_id);
CREATE INDEX idx_connections_status ON presence.connections(status);
CREATE INDEX idx_connections_heartbeat ON presence.connections(last_heartbeat_at)
    WHERE status = 'connected';
CREATE INDEX idx_activity_log_user ON presence.activity_log(user_id, created_at DESC);
CREATE INDEX idx_activity_log_context ON presence.activity_log(context_type, context_id);
CREATE INDEX idx_typing_indicators_conversation ON presence.typing_indicators(conversation_id);
CREATE INDEX idx_typing_indicators_expires ON presence.typing_indicators(expires_at);
CREATE INDEX idx_subscriptions_subscriber ON presence.subscriptions(subscriber_user_id)
    WHERE is_active = TRUE;
CREATE INDEX idx_subscriptions_target ON presence.subscriptions(target_user_id)
    WHERE is_active = TRUE;
CREATE INDEX idx_status_history_user ON presence.status_history(user_id, changed_at DESC);

-- Functions

-- Function: Update user presence status
CREATE OR REPLACE FUNCTION presence.update_user_status(
    p_user_id UUID,
    p_status VARCHAR(20),
    p_device_id VARCHAR(255) DEFAULT NULL
) RETURNS VOID AS $$
DECLARE
    v_prev_status VARCHAR(20);
    v_prev_last_seen TIMESTAMPTZ;
BEGIN
    -- Get previous status
    SELECT status, last_seen_at INTO v_prev_status, v_prev_last_seen
    FROM presence.user_status
    WHERE user_id = p_user_id;

    -- Update or insert status
    INSERT INTO presence.user_status (
        user_id, status, last_seen_at, last_activity_at,
        is_online, primary_device_id, updated_at
    )
    VALUES (
        p_user_id, p_status, NOW(), NOW(),
        (p_status = 'online'), p_device_id, NOW()
    )
    ON CONFLICT (user_id) DO UPDATE SET
        status = EXCLUDED.status,
        last_seen_at = NOW(),
        last_activity_at = NOW(),
        is_online = (EXCLUDED.status = 'online'),
        primary_device_id = COALESCE(EXCLUDED.primary_device_id, presence.user_status.primary_device_id),
        updated_at = NOW();

    -- Log status change if different
    IF v_prev_status IS NOT NULL AND v_prev_status != p_status THEN
        INSERT INTO presence.status_history (
            user_id, previous_status, new_status,
            device_id, duration_seconds, change_reason
        )
        VALUES (
            p_user_id, v_prev_status, p_status,
            p_device_id,
            EXTRACT(EPOCH FROM (NOW() - v_prev_last_seen))::INTEGER,
            'manual'
        );
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function: Clean up expired typing indicators
CREATE OR REPLACE FUNCTION presence.cleanup_expired_typing() RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM presence.typing_indicators
    WHERE expires_at < NOW();

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function: Mark stale connections as disconnected
CREATE OR REPLACE FUNCTION presence.cleanup_stale_connections() RETURNS INTEGER AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    -- Mark connections stale if no heartbeat for 30 seconds
    UPDATE presence.connections
    SET
        status = 'disconnected',
        disconnected_at = NOW(),
        updated_at = NOW()
    WHERE
        status = 'connected'
        AND last_heartbeat_at < NOW() - INTERVAL '30 seconds';

    GET DIAGNOSTICS v_updated_count = ROW_COUNT;

    -- Update user status if no active connections
    UPDATE presence.user_status us
    SET
        status = 'offline',
        is_online = FALSE,
        last_seen_at = NOW(),
        updated_at = NOW()
    WHERE
        is_online = TRUE
        AND NOT EXISTS (
            SELECT 1 FROM presence.connections c
            WHERE c.user_id = us.user_id
            AND c.status = 'connected'
        );

    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql;

-- Trigger: Update connection count when connections change
CREATE OR REPLACE FUNCTION presence.update_connection_count() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        UPDATE presence.user_status
        SET
            active_connections = (
                SELECT COUNT(*)
                FROM presence.connections
                WHERE user_id = NEW.user_id AND status = 'connected'
            ),
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE presence.user_status
        SET
            active_connections = (
                SELECT COUNT(*)
                FROM presence.connections
                WHERE user_id = OLD.user_id AND status = 'connected'
            ),
            updated_at = NOW()
        WHERE user_id = OLD.user_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_connection_count
AFTER INSERT OR UPDATE OR DELETE ON presence.connections
FOR EACH ROW EXECUTE FUNCTION presence.update_connection_count();

-- Track migration
INSERT INTO schema_migrations (version, description)
VALUES (2, 'Add presence schema for real-time user presence tracking')
ON CONFLICT (version) DO NOTHING;
