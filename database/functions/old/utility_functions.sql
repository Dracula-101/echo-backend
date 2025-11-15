-- =====================================================
-- ROW LEVEL SECURITY (RLS) & UTILITY FUNCTIONS
-- =====================================================

-- =====================================================
-- UTILITY FUNCTIONS
-- =====================================================

-- Get current authenticated user ID (from JWT or session context)
CREATE OR REPLACE FUNCTION auth.current_user_id()
RETURNS UUID AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_user_id', TRUE), '')::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user is admin
CREATE OR REPLACE FUNCTION auth.is_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN COALESCE(
        (SELECT metadata->>'is_admin' = 'true' 
         FROM auth.users 
         WHERE id = auth.current_user_id()),
        FALSE
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user is moderator
CREATE OR REPLACE FUNCTION auth.is_moderator()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN COALESCE(
        (SELECT metadata->>'is_moderator' = 'true' 
         FROM auth.users 
         WHERE id = auth.current_user_id()),
        FALSE
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user has specific role in conversation
CREATE OR REPLACE FUNCTION messages.user_has_conversation_role(
    conv_id UUID,
    required_role VARCHAR
)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = conv_id
        AND user_id = auth.current_user_id()
        AND role = required_role
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user is conversation participant
CREATE OR REPLACE FUNCTION messages.is_conversation_participant(conv_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = conv_id
        AND user_id = auth.current_user_id()
        AND left_at IS NULL
        AND removed_at IS NULL
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if users are contacts/friends
CREATE OR REPLACE FUNCTION users.are_contacts(user_id_a UUID, user_id_b UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM users.contacts
        WHERE ((user_id = user_id_a AND contact_user_id = user_id_b)
           OR (user_id = user_id_b AND contact_user_id = user_id_a))
        AND status = 'accepted'
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user is blocked
CREATE OR REPLACE FUNCTION users.is_blocked_by(target_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM users.blocked_users
        WHERE user_id = target_user_id
        AND blocked_user_id = auth.current_user_id()
        AND unblocked_at IS NULL
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if current user blocked someone
CREATE OR REPLACE FUNCTION users.has_blocked(target_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM users.blocked_users
        WHERE user_id = auth.current_user_id()
        AND blocked_user_id = target_user_id
        AND unblocked_at IS NULL
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Generate unique username
CREATE OR REPLACE FUNCTION users.generate_unique_username(base_name VARCHAR)
RETURNS VARCHAR AS $$
DECLARE
    username VARCHAR;
    counter INTEGER := 0;
BEGIN
    username := LOWER(REGEXP_REPLACE(base_name, '[^a-zA-Z0-9_]', '', 'g'));
    
    WHILE EXISTS(SELECT 1 FROM users.profiles WHERE profiles.username = username) LOOP
        counter := counter + 1;
        username := LOWER(REGEXP_REPLACE(base_name, '[^a-zA-Z0-9_]', '', 'g')) || counter::TEXT;
    END LOOP;
    
    RETURN username;
END;
$$ LANGUAGE plpgsql;

-- Calculate distance between two points (Haversine formula)
CREATE OR REPLACE FUNCTION location.calculate_distance(
    lat1 DECIMAL, lon1 DECIMAL,
    lat2 DECIMAL, lon2 DECIMAL
)
RETURNS DECIMAL AS $$
DECLARE
    earth_radius DECIMAL := 6371; -- km
    dlat DECIMAL;
    dlon DECIMAL;
    a DECIMAL;
    c DECIMAL;
BEGIN
    dlat := RADIANS(lat2 - lat1);
    dlon := RADIANS(lon2 - lon1);
    
    a := SIN(dlat/2) * SIN(dlat/2) + 
         COS(RADIANS(lat1)) * COS(RADIANS(lat2)) * 
         SIN(dlon/2) * SIN(dlon/2);
    
    c := 2 * ATAN2(SQRT(a), SQRT(1-a));
    
    RETURN earth_radius * c;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Hash password with salt
CREATE OR REPLACE FUNCTION auth.hash_password(password TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Using pgcrypto extension
    RETURN crypt(password, gen_salt('bf', 10));
END;
$$ LANGUAGE plpgsql;

-- Verify password
CREATE OR REPLACE FUNCTION auth.verify_password(password TEXT, hash TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN hash = crypt(password, hash);
END;
$$ LANGUAGE plpgsql;

-- Generate OTP code
CREATE OR REPLACE FUNCTION auth.generate_otp(length INTEGER DEFAULT 6)
RETURNS VARCHAR AS $$
DECLARE
    otp VARCHAR;
    i INTEGER;
BEGIN
    otp := '';
    FOR i IN 1..length LOOP
        otp := otp || FLOOR(RANDOM() * 10)::TEXT;
    END LOOP;
    RETURN otp;
END;
$$ LANGUAGE plpgsql;

-- Generate secure token
CREATE OR REPLACE FUNCTION auth.generate_token(length INTEGER DEFAULT 32)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(gen_random_bytes(length), 'hex');
END;
$$ LANGUAGE plpgsql;

-- Format file size to human readable
CREATE OR REPLACE FUNCTION media.format_file_size(bytes BIGINT)
RETURNS TEXT AS $$
DECLARE
    kb DECIMAL := 1024;
    mb DECIMAL := 1024 * 1024;
    gb DECIMAL := 1024 * 1024 * 1024;
BEGIN
    IF bytes < kb THEN
        RETURN bytes || ' B';
    ELSIF bytes < mb THEN
        RETURN ROUND(bytes / kb, 2) || ' KB';
    ELSIF bytes < gb THEN
        RETURN ROUND(bytes / mb, 2) || ' MB';
    ELSE
        RETURN ROUND(bytes / gb, 2) || ' GB';
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Search messages with full-text search
CREATE OR REPLACE FUNCTION messages.search_messages(
    search_query TEXT,
    conv_id UUID DEFAULT NULL,
    limit_count INTEGER DEFAULT 50
)
RETURNS TABLE (
    message_id UUID,
    conversation_id UUID,
    content TEXT,
    sender_user_id UUID,
    created_at TIMESTAMPTZ,
    rank REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id,
        m.conversation_id,
        m.content,
        m.sender_user_id,
        m.created_at,
        ts_rank(si.content_tsvector, plainto_tsquery('english', search_query)) AS rank
    FROM messages.messages m
    JOIN messages.search_index si ON si.message_id = m.id
    WHERE si.content_tsvector @@ plainto_tsquery('english', search_query)
    AND (conv_id IS NULL OR m.conversation_id = conv_id)
    AND messages.is_conversation_participant(m.conversation_id)
    AND m.is_deleted = FALSE
    ORDER BY rank DESC, m.created_at DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Get user's conversation list with unread counts
CREATE OR REPLACE FUNCTION messages.get_user_conversations(user_uuid UUID)
RETURNS TABLE (
    conversation_id UUID,
    conversation_type VARCHAR,
    title VARCHAR,
    avatar_url TEXT,
    last_message TEXT,
    last_message_at TIMESTAMPTZ,
    unread_count INTEGER,
    is_muted BOOLEAN,
    is_pinned BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        c.id,
        c.conversation_type,
        c.title,
        c.avatar_url,
        m.content,
        c.last_message_at,
        cp.unread_count,
        cp.is_muted,
        cp.is_pinned
    FROM messages.conversations c
    JOIN messages.conversation_participants cp ON cp.conversation_id = c.id
    LEFT JOIN messages.messages m ON m.id = c.last_message_id
    WHERE cp.user_id = user_uuid
    AND cp.left_at IS NULL
    AND cp.removed_at IS NULL
    AND c.is_active = TRUE
    ORDER BY cp.is_pinned DESC, c.last_message_at DESC NULLS LAST;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- =====================================================
-- ROW LEVEL SECURITY POLICIES
-- =====================================================

-- Enable RLS on all tables
ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.security_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.login_history ENABLE ROW LEVEL SECURITY;

ALTER TABLE users.profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.blocked_users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.status_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.activity_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.devices ENABLE ROW LEVEL SECURITY;

ALTER TABLE messages.conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.conversation_participants ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.reactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.delivery_status ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.bookmarks ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.drafts ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.calls ENABLE ROW LEVEL SECURITY;

ALTER TABLE media.files ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.albums ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.shares ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.storage_stats ENABLE ROW LEVEL SECURITY;

ALTER TABLE notifications.notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.user_preferences ENABLE ROW LEVEL SECURITY;

ALTER TABLE analytics.events ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics.user_sessions ENABLE ROW LEVEL SECURITY;

ALTER TABLE location.user_locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE location.location_shares ENABLE ROW LEVEL SECURITY;

-- =====================================================
-- AUTH SCHEMA POLICIES
-- =====================================================

-- Users can read their own user data
CREATE POLICY users_select_own ON auth.users
    FOR SELECT
    USING (id = auth.current_user_id());

-- Users can update their own data
CREATE POLICY users_update_own ON auth.users
    FOR UPDATE
    USING (id = auth.current_user_id());

-- Admins can see all users
CREATE POLICY users_admin_all ON auth.users
    FOR ALL
    USING (auth.is_admin());

-- Users can see their own sessions
CREATE POLICY sessions_select_own ON auth.sessions
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can delete their own sessions
CREATE POLICY sessions_delete_own ON auth.sessions
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Users can see their own security events
CREATE POLICY security_events_select_own ON auth.security_events
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can see their own login history
CREATE POLICY login_history_select_own ON auth.login_history
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- =====================================================
-- USERS SCHEMA POLICIES
-- =====================================================

-- Public profiles are visible to everyone
CREATE POLICY profiles_select_public ON users.profiles
    FOR SELECT
    USING (
        profile_visibility = 'public'
        OR user_id = auth.current_user_id()
        OR (profile_visibility = 'friends' AND users.are_contacts(user_id, auth.current_user_id()))
    );

-- Users can update their own profile
CREATE POLICY profiles_update_own ON users.profiles
    FOR UPDATE
    USING (user_id = auth.current_user_id());

-- Users can insert their own profile
CREATE POLICY profiles_insert_own ON users.profiles
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

-- Users can see their own contacts
CREATE POLICY contacts_select_own ON users.contacts
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR contact_user_id = auth.current_user_id()
    );

-- Users can manage their own contacts
CREATE POLICY contacts_insert_own ON users.contacts
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

CREATE POLICY contacts_update_own ON users.contacts
    FOR UPDATE
    USING (user_id = auth.current_user_id());

CREATE POLICY contacts_delete_own ON users.contacts
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Users can only access their own settings
CREATE POLICY settings_own_only ON users.settings
    FOR ALL
    USING (user_id = auth.current_user_id());

-- Users can manage their own blocked users
CREATE POLICY blocked_users_own_only ON users.blocked_users
    FOR ALL
    USING (user_id = auth.current_user_id());

-- Status history visibility based on privacy settings
CREATE POLICY status_history_select ON users.status_history
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR privacy = 'public'
        OR (privacy = 'contacts' AND users.are_contacts(user_id, auth.current_user_id()))
        AND expires_at > NOW()
        AND deleted_at IS NULL
    );

-- Users can manage their own status
CREATE POLICY status_history_manage_own ON users.status_history
    FOR ALL
    USING (user_id = auth.current_user_id());

-- Users can only see their own activity log
CREATE POLICY activity_log_own_only ON users.activity_log
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can manage their own devices
CREATE POLICY devices_own_only ON users.devices
    FOR ALL
    USING (user_id = auth.current_user_id());

-- =====================================================
-- MESSAGES SCHEMA POLICIES
-- =====================================================

-- Users can see conversations they're part of
CREATE POLICY conversations_select_participant ON messages.conversations
    FOR SELECT
    USING (messages.is_conversation_participant(id));

-- Users can create conversations
CREATE POLICY conversations_insert ON messages.conversations
    FOR INSERT
    WITH CHECK (creator_user_id = auth.current_user_id());

-- Owners and admins can update conversations
CREATE POLICY conversations_update ON messages.conversations
    FOR UPDATE
    USING (
        creator_user_id = auth.current_user_id()
        OR messages.user_has_conversation_role(id, 'admin')
        OR messages.user_has_conversation_role(id, 'owner')
    );

-- Users can see participants in their conversations
CREATE POLICY participants_select ON messages.conversation_participants
    FOR SELECT
    USING (messages.is_conversation_participant(conversation_id));

-- Admins can manage participants
CREATE POLICY participants_manage ON messages.conversation_participants
    FOR ALL
    USING (
        user_id = auth.current_user_id()
        OR messages.user_has_conversation_role(conversation_id, 'admin')
        OR messages.user_has_conversation_role(conversation_id, 'owner')
    );

-- Users can see messages in their conversations
CREATE POLICY messages_select_participant ON messages.messages
    FOR SELECT
    USING (
        messages.is_conversation_participant(conversation_id)
        AND is_deleted = FALSE
        AND NOT users.is_blocked_by(sender_user_id)
    );

-- Users can send messages to their conversations
CREATE POLICY messages_insert ON messages.messages
    FOR INSERT
    WITH CHECK (
        sender_user_id = auth.current_user_id()
        AND messages.is_conversation_participant(conversation_id)
        AND (SELECT can_send_messages FROM messages.conversation_participants 
             WHERE conversation_id = messages.conversation_id 
             AND user_id = auth.current_user_id())
    );

-- Users can update their own messages
CREATE POLICY messages_update_own ON messages.messages
    FOR UPDATE
    USING (sender_user_id = auth.current_user_id());

-- Users can delete their own messages or admins can delete any
CREATE POLICY messages_delete ON messages.messages
    FOR DELETE
    USING (
        sender_user_id = auth.current_user_id()
        OR messages.user_has_conversation_role(conversation_id, 'admin')
        OR messages.user_has_conversation_role(conversation_id, 'moderator')
    );

-- Users can see reactions in their conversations
CREATE POLICY reactions_select ON messages.reactions
    FOR SELECT
    USING (
        EXISTS(
            SELECT 1 FROM messages.messages m
            WHERE m.id = message_id
            AND messages.is_conversation_participant(m.conversation_id)
        )
    );

-- Users can manage their own reactions
CREATE POLICY reactions_manage_own ON messages.reactions
    FOR ALL
    USING (user_id = auth.current_user_id());

-- Users can see delivery status for their messages
CREATE POLICY delivery_status_select ON messages.delivery_status
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR EXISTS(
            SELECT 1 FROM messages.messages m
            WHERE m.id = message_id
            AND m.sender_user_id = auth.current_user_id()
        )
    );

-- Users can manage their own bookmarks
CREATE POLICY bookmarks_own_only ON messages.bookmarks
    FOR ALL
    USING (user_id = auth.current_user_id());

-- Users can manage their own drafts
CREATE POLICY drafts_own_only ON messages.drafts
    FOR ALL
    USING (user_id = auth.current_user_id());

-- Users can see calls in their conversations
CREATE POLICY calls_select ON messages.calls
    FOR SELECT
    USING (
        messages.is_conversation_participant(conversation_id)
        OR initiator_user_id = auth.current_user_id()
    );

-- =====================================================
-- MEDIA SCHEMA POLICIES
-- =====================================================

-- Users can see their own files and shared files
CREATE POLICY files_select ON media.files
    FOR SELECT
    USING (
        uploader_user_id = auth.current_user_id()
        OR visibility = 'public'
        OR EXISTS(
            SELECT 1 FROM media.shares s
            WHERE s.file_id = id
            AND s.shared_with_user_id = auth.current_user_id()
            AND s.is_active = TRUE
        )
        OR EXISTS(
            SELECT 1 FROM messages.message_media mm
            JOIN messages.messages m ON m.id = mm.message_id
            WHERE mm.media_id = id
            AND messages.is_conversation_participant(m.conversation_id)
        )
    );

-- Users can upload files
CREATE POLICY files_insert ON media.files
    FOR INSERT
    WITH CHECK (uploader_user_id = auth.current_user_id());

-- Users can update their own files
CREATE POLICY files_update_own ON media.files
    FOR UPDATE
    USING (uploader_user_id = auth.current_user_id());

-- Users can delete their own files
CREATE POLICY files_delete_own ON media.files
    FOR DELETE
    USING (uploader_user_id = auth.current_user_id());

-- Users can manage their own albums
CREATE POLICY albums_own_only ON media.albums
    FOR ALL
    USING (user_id = auth.current_user_id());

-- Users can manage their own shares
CREATE POLICY shares_manage_own ON media.shares
    FOR ALL
    USING (
        shared_by_user_id = auth.current_user_id()
        OR shared_with_user_id = auth.current_user_id()
    );

-- Users can see their own storage stats
CREATE POLICY storage_stats_own_only ON media.storage_stats
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- =====================================================
-- NOTIFICATIONS SCHEMA POLICIES
-- =====================================================

-- Users can only see their own notifications
CREATE POLICY notifications_own_only ON notifications.notifications
    FOR ALL
    USING (user_id = auth.current_user_id());

-- Users can only manage their own notification preferences
CREATE POLICY notification_prefs_own_only ON notifications.user_preferences
    FOR ALL
    USING (user_id = auth.current_user_id());

-- =====================================================
-- ANALYTICS SCHEMA POLICIES
-- =====================================================

-- Users can see their own events
CREATE POLICY events_own_only ON analytics.events
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can see their own sessions
CREATE POLICY user_sessions_own_only ON analytics.user_sessions
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Admins and analysts can see all analytics
CREATE POLICY analytics_admin_access ON analytics.events
    FOR ALL
    USING (auth.is_admin() OR auth.is_moderator());

-- =====================================================
-- LOCATION SCHEMA POLICIES
-- =====================================================

-- Users can see their own location history
CREATE POLICY user_locations_own_only ON location.user_locations
    FOR ALL
    USING (user_id = auth.current_user_id());

-- Users can see location shares shared with them
CREATE POLICY location_shares_select ON location.location_shares
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR shared_with_user_id = auth.current_user_id()
    );

-- Users can manage their own location shares
CREATE POLICY location_shares_manage_own ON location.location_shares
    FOR ALL
    USING (user_id = auth.current_user_id());

-- =====================================================
-- GRANT PERMISSIONS
-- =====================================================

-- Create application role for authenticated users
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user;
    END IF;
END
$$;

-- Grant usage on schemas
GRANT USAGE ON SCHEMA auth TO app_user;
GRANT USAGE ON SCHEMA users TO app_user;
GRANT USAGE ON SCHEMA messages TO app_user;
GRANT USAGE ON SCHEMA media TO app_user;
GRANT USAGE ON SCHEMA notifications TO app_user;
GRANT USAGE ON SCHEMA analytics TO app_user;
GRANT USAGE ON SCHEMA location TO app_user;

-- Grant table permissions (SELECT, INSERT, UPDATE, DELETE where appropriate)
GRANT SELECT, INSERT, UPDATE ON auth.users TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON auth.sessions TO app_user;
GRANT SELECT ON auth.security_events TO app_user;
GRANT SELECT ON auth.login_history TO app_user;

GRANT SELECT, INSERT, UPDATE ON users.profiles TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON users.contacts TO app_user;
GRANT SELECT, INSERT, UPDATE ON users.settings TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON users.blocked_users TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON users.status_history TO app_user;
GRANT SELECT ON users.activity_log TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON users.devices TO app_user;

GRANT SELECT, INSERT, UPDATE ON messages.conversations TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON messages.conversation_participants TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON messages.messages TO app_user;
GRANT SELECT, INSERT, DELETE ON messages.reactions TO app_user;
GRANT SELECT ON messages.delivery_status TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON messages.bookmarks TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON messages.drafts TO app_user;
GRANT SELECT, INSERT ON messages.calls TO app_user;

GRANT SELECT, INSERT, UPDATE, DELETE ON media.files TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON media.albums TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON media.shares TO app_user;
GRANT SELECT ON media.storage_stats TO app_user;

GRANT SELECT, UPDATE ON notifications.notifications TO app_user;
GRANT SELECT, INSERT, UPDATE ON notifications.user_preferences TO app_user;

GRANT SELECT ON analytics.events TO app_user;
GRANT SELECT ON analytics.user_sessions TO app_user;

GRANT SELECT, INSERT ON location.user_locations TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON location.location_shares TO app_user;

-- Grant execute on utility functions
GRANT EXECUTE ON FUNCTION auth.current_user_id() TO app_user;
GRANT EXECUTE ON FUNCTION auth.is_admin() TO app_user;
GRANT EXECUTE ON FUNCTION messages.is_conversation_participant(UUID) TO app_user;
GRANT EXECUTE ON FUNCTION users.are_contacts(UUID, UUID) TO app_user;
GRANT EXECUTE ON FUNCTION messages.search_messages(TEXT, UUID, INTEGER) TO app_user;
GRANT EXECUTE ON FUNCTION messages.get_user_conversations(UUID) TO app_user;