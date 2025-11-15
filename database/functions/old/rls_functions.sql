-- =====================================================
-- ROW LEVEL SECURITY (RLS) & UTILITY FUNCTIONS
-- =====================================================

-- =====================================================
-- CORE UTILITY FUNCTIONS
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

-- Check if account is active
CREATE OR REPLACE FUNCTION auth.is_account_active()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN COALESCE(
        (SELECT account_status = 'active'
         FROM auth.users 
         WHERE id = auth.current_user_id()),
        FALSE
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- =====================================================
-- MESSAGE & CONVERSATION RLS FUNCTIONS
-- =====================================================

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

-- Check if user is conversation owner
CREATE OR REPLACE FUNCTION messages.is_conversation_owner(conv_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM messages.conversations
        WHERE id = conv_id
        AND creator_user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user is conversation admin or owner
CREATE OR REPLACE FUNCTION messages.is_conversation_admin(conv_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = conv_id
        AND user_id = auth.current_user_id()
        AND role IN ('owner', 'admin')
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user can send messages in conversation
CREATE OR REPLACE FUNCTION messages.can_send_messages(conv_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = conv_id
        AND user_id = auth.current_user_id()
        AND can_send_messages = TRUE
        AND left_at IS NULL
        AND removed_at IS NULL
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user can delete message
CREATE OR REPLACE FUNCTION messages.can_delete_message(msg_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    msg_conv_id UUID;
    msg_sender_id UUID;
BEGIN
    SELECT conversation_id, sender_user_id INTO msg_conv_id, msg_sender_id
    FROM messages.messages
    WHERE id = msg_id;
    
    -- Can delete own messages or if admin/moderator
    RETURN msg_sender_id = auth.current_user_id()
        OR messages.is_conversation_admin(msg_conv_id)
        OR EXISTS(
            SELECT 1 FROM messages.conversation_participants
            WHERE conversation_id = msg_conv_id
            AND user_id = auth.current_user_id()
            AND role = 'moderator'
        );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user owns message
CREATE OR REPLACE FUNCTION messages.is_message_owner(msg_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM messages.messages
        WHERE id = msg_id
        AND sender_user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if conversation is public
CREATE OR REPLACE FUNCTION messages.is_conversation_public(conv_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN COALESCE(
        (SELECT is_public FROM messages.conversations WHERE id = conv_id),
        FALSE
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- =====================================================
-- USER & PROFILE RLS FUNCTIONS
-- =====================================================

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

-- Check if current user is contacts with target
CREATE OR REPLACE FUNCTION users.is_contact_with(target_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN users.are_contacts(auth.current_user_id(), target_user_id);
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user is blocked by someone
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

-- Check if users have mutual block
CREATE OR REPLACE FUNCTION users.has_mutual_block(user_id_a UUID, user_id_b UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM users.blocked_users
        WHERE ((user_id = user_id_a AND blocked_user_id = user_id_b)
           OR (user_id = user_id_b AND blocked_user_id = user_id_a))
        AND unblocked_at IS NULL
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check profile visibility for current user
CREATE OR REPLACE FUNCTION users.can_view_profile(target_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    visibility VARCHAR(20);
BEGIN
    -- Own profile is always visible
    IF target_user_id = auth.current_user_id() THEN
        RETURN TRUE;
    END IF;
    
    -- Check if blocked
    IF users.has_mutual_block(auth.current_user_id(), target_user_id) THEN
        RETURN FALSE;
    END IF;
    
    -- Get profile visibility setting
    SELECT profile_visibility INTO visibility
    FROM users.profiles
    WHERE user_id = target_user_id;
    
    -- Check visibility rules
    RETURN CASE visibility
        WHEN 'public' THEN TRUE
        WHEN 'friends' THEN users.are_contacts(auth.current_user_id(), target_user_id)
        WHEN 'private' THEN FALSE
        ELSE FALSE
    END;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user can see online status
CREATE OR REPLACE FUNCTION users.can_view_online_status(target_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    visibility VARCHAR(20);
    override_visible BOOLEAN;
BEGIN
    -- Own status is always visible
    IF target_user_id = auth.current_user_id() THEN
        RETURN TRUE;
    END IF;
    
    -- Check for privacy override
    SELECT online_status_visible INTO override_visible
    FROM users.privacy_overrides
    WHERE user_id = target_user_id
    AND target_user_id = auth.current_user_id();
    
    IF override_visible IS NOT NULL THEN
        RETURN override_visible;
    END IF;
    
    -- Get default visibility from settings
    SELECT online_status_visibility INTO visibility
    FROM users.settings
    WHERE user_id = target_user_id;
    
    RETURN CASE visibility
        WHEN 'everyone' THEN TRUE
        WHEN 'contacts' THEN users.are_contacts(auth.current_user_id(), target_user_id)
        WHEN 'nobody' THEN FALSE
        ELSE FALSE
    END;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user can see last seen timestamp
CREATE OR REPLACE FUNCTION users.can_view_last_seen(target_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    visibility VARCHAR(20);
    override_visible BOOLEAN;
BEGIN
    IF target_user_id = auth.current_user_id() THEN
        RETURN TRUE;
    END IF;
    
    -- Check for privacy override
    SELECT last_seen_visible INTO override_visible
    FROM users.privacy_overrides
    WHERE user_id = target_user_id
    AND target_user_id = auth.current_user_id();
    
    IF override_visible IS NOT NULL THEN
        RETURN override_visible;
    END IF;
    
    SELECT last_seen_visibility INTO visibility
    FROM users.settings
    WHERE user_id = target_user_id;
    
    RETURN CASE visibility
        WHEN 'everyone' THEN TRUE
        WHEN 'contacts' THEN users.are_contacts(auth.current_user_id(), target_user_id)
        WHEN 'nobody' THEN FALSE
        ELSE FALSE
    END;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user can view status updates
CREATE OR REPLACE FUNCTION users.can_view_status(status_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    status_user_id UUID;
    status_privacy VARCHAR(20);
BEGIN
    SELECT user_id, privacy INTO status_user_id, status_privacy
    FROM users.status_history
    WHERE id = status_id;
    
    -- Own status is always visible
    IF status_user_id = auth.current_user_id() THEN
        RETURN TRUE;
    END IF;
    
    -- Check blocked status
    IF users.has_mutual_block(auth.current_user_id(), status_user_id) THEN
        RETURN FALSE;
    END IF;
    
    RETURN CASE status_privacy
        WHEN 'public' THEN TRUE
        WHEN 'contacts' THEN users.are_contacts(auth.current_user_id(), status_user_id)
        WHEN 'close_friends' THEN EXISTS(
            SELECT 1 FROM users.contacts
            WHERE user_id = status_user_id
            AND contact_user_id = auth.current_user_id()
            AND is_favorite = TRUE
            AND status = 'accepted'
        )
        WHEN 'private' THEN FALSE
        ELSE FALSE
    END;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- =====================================================
-- MEDIA & FILE RLS FUNCTIONS
-- =====================================================

-- Check if user owns file
CREATE OR REPLACE FUNCTION media.is_file_owner(file_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM media.files
        WHERE id = file_id
        AND uploader_user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if file is shared with user
CREATE OR REPLACE FUNCTION media.is_file_shared_with_user(file_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM media.shares
        WHERE file_id = file_id
        AND (shared_with_user_id = auth.current_user_id()
             OR shared_with_conversation_id IN (
                 SELECT conversation_id FROM messages.conversation_participants
                 WHERE user_id = auth.current_user_id()
             ))
        AND is_active = TRUE
        AND (expires_at IS NULL OR expires_at > NOW())
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if file is in accessible message
CREATE OR REPLACE FUNCTION media.is_file_in_accessible_message(file_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM messages.message_media mm
        JOIN messages.messages m ON m.id = mm.message_id
        WHERE mm.media_id = file_id
        AND messages.is_conversation_participant(m.conversation_id)
        AND m.is_deleted = FALSE
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user can access file
CREATE OR REPLACE FUNCTION media.can_access_file(file_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    file_visibility VARCHAR(50);
BEGIN
    -- Check if owner
    IF media.is_file_owner(file_id) THEN
        RETURN TRUE;
    END IF;
    
    -- Check visibility
    SELECT visibility INTO file_visibility
    FROM media.files
    WHERE id = file_id;
    
    IF file_visibility = 'public' THEN
        RETURN TRUE;
    END IF;
    
    -- Check if shared
    IF media.is_file_shared_with_user(file_id) THEN
        RETURN TRUE;
    END IF;
    
    -- Check if in accessible message
    IF media.is_file_in_accessible_message(file_id) THEN
        RETURN TRUE;
    END IF;
    
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user owns album
CREATE OR REPLACE FUNCTION media.is_album_owner(album_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM media.albums
        WHERE id = album_id
        AND user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- =====================================================
-- NOTIFICATION RLS FUNCTIONS
-- =====================================================

-- Check if notification belongs to user
CREATE OR REPLACE FUNCTION notifications.is_notification_owner(notification_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM notifications.notifications
        WHERE id = notification_id
        AND user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user can view announcement
CREATE OR REPLACE FUNCTION notifications.can_view_announcement(announcement_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    target_audience VARCHAR(100);
    target_countries VARCHAR(5)[];
    user_country VARCHAR(5);
BEGIN
    SELECT 
        a.target_audience,
        a.target_countries
    INTO 
        target_audience,
        target_countries
    FROM notifications.announcements a
    WHERE a.id = announcement_id
    AND a.is_active = TRUE
    AND (a.starts_at IS NULL OR a.starts_at <= NOW())
    AND (a.ends_at IS NULL OR a.ends_at > NOW());
    
    IF target_audience = 'all' THEN
        RETURN TRUE;
    END IF;
    
    -- Check country targeting
    IF target_countries IS NOT NULL AND array_length(target_countries, 1) > 0 THEN
        SELECT country_code INTO user_country
        FROM users.profiles
        WHERE user_id = auth.current_user_id();
        
        IF user_country IS NULL OR NOT (user_country = ANY(target_countries)) THEN
            RETURN FALSE;
        END IF;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- =====================================================
-- LOCATION RLS FUNCTIONS
-- =====================================================

-- Check if user owns location record
CREATE OR REPLACE FUNCTION location.is_location_owner(location_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM location.user_locations
        WHERE id = location_id
        AND user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if location is shared with user
CREATE OR REPLACE FUNCTION location.is_location_shared_with_user(share_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM location.location_shares
        WHERE id = share_id
        AND (shared_with_user_id = auth.current_user_id()
             OR shared_with_conversation_id IN (
                 SELECT conversation_id FROM messages.conversation_participants
                 WHERE user_id = auth.current_user_id()
             ))
        AND is_active = TRUE
        AND (expires_at IS NULL OR expires_at > NOW())
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- =====================================================
-- ANALYTICS RLS FUNCTIONS
-- =====================================================

-- Check if user can view analytics
CREATE OR REPLACE FUNCTION analytics.can_view_analytics(target_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN target_user_id = auth.current_user_id()
        OR auth.is_admin()
        OR auth.is_moderator();
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user owns event
CREATE OR REPLACE FUNCTION analytics.is_event_owner(event_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM analytics.events
        WHERE id = event_id
        AND user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- =====================================================
-- CALL & MEETING RLS FUNCTIONS
-- =====================================================

-- Check if user is call participant
CREATE OR REPLACE FUNCTION messages.is_call_participant(call_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM messages.call_participants
        WHERE call_id = call_id
        AND user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Check if user initiated call
CREATE OR REPLACE FUNCTION messages.is_call_initiator(call_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM messages.calls
        WHERE id = call_id
        AND initiator_user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- =====================================================
-- ADDITIONAL UTILITY FUNCTIONS
-- =====================================================

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