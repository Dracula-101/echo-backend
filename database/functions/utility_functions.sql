-- =====================================================
-- Utility Functions for Common Operations
-- =====================================================

-- Generate a random slug
CREATE OR REPLACE FUNCTION public.generate_slug(input_text TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN LOWER(
        REGEXP_REPLACE(
            REGEXP_REPLACE(input_text, '[^a-zA-Z0-9\s-]', '', 'g'),
            '\s+', '-', 'g'
        )
    ) || '-' || SUBSTRING(MD5(RANDOM()::TEXT) FROM 1 FOR 8);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Generate a secure random token
CREATE OR REPLACE FUNCTION public.generate_token(length INTEGER DEFAULT 32)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(gen_random_bytes(length), 'base64');
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Generate a numeric OTP code
CREATE OR REPLACE FUNCTION auth.generate_otp_code(digits INTEGER DEFAULT 6)
RETURNS TEXT AS $$
BEGIN
    RETURN LPAD(FLOOR(RANDOM() * POWER(10, digits))::TEXT, digits, '0');
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Calculate distance between two coordinates (in kilometers)
CREATE OR REPLACE FUNCTION public.calculate_distance(
    lat1 DECIMAL,
    lon1 DECIMAL,
    lat2 DECIMAL,
    lon2 DECIMAL
)
RETURNS DECIMAL AS $$
DECLARE
    earth_radius CONSTANT DECIMAL := 6371; -- km
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

-- Format file size to human readable
CREATE OR REPLACE FUNCTION public.format_file_size(bytes BIGINT)
RETURNS TEXT AS $$
DECLARE
    units TEXT[] := ARRAY['B', 'KB', 'MB', 'GB', 'TB'];
    size DECIMAL := bytes;
    unit_index INTEGER := 1;
BEGIN
    WHILE size >= 1024 AND unit_index < ARRAY_LENGTH(units, 1) LOOP
        size := size / 1024.0;
        unit_index := unit_index + 1;
    END LOOP;
    
    RETURN ROUND(size, 2)::TEXT || ' ' || units[unit_index];
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Get time ago string
CREATE OR REPLACE FUNCTION public.time_ago(timestamp_input TIMESTAMPTZ)
RETURNS TEXT AS $$
DECLARE
    diff INTERVAL;
    seconds INTEGER;
    minutes INTEGER;
    hours INTEGER;
    days INTEGER;
BEGIN
    diff := NOW() - timestamp_input;
    seconds := EXTRACT(EPOCH FROM diff)::INTEGER;
    
    IF seconds < 60 THEN
        RETURN 'just now';
    ELSIF seconds < 3600 THEN
        minutes := seconds / 60;
        RETURN minutes || ' minute' || (CASE WHEN minutes > 1 THEN 's' ELSE '' END) || ' ago';
    ELSIF seconds < 86400 THEN
        hours := seconds / 3600;
        RETURN hours || ' hour' || (CASE WHEN hours > 1 THEN 's' ELSE '' END) || ' ago';
    ELSIF seconds < 604800 THEN
        days := seconds / 86400;
        RETURN days || ' day' || (CASE WHEN days > 1 THEN 's' ELSE '' END) || ' ago';
    ELSE
        RETURN TO_CHAR(timestamp_input, 'Mon DD, YYYY');
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Search users by username or email
CREATE OR REPLACE FUNCTION users.search_users(
    search_query TEXT,
    limit_count INTEGER DEFAULT 20,
    offset_count INTEGER DEFAULT 0
)
RETURNS TABLE (
    user_id UUID,
    username VARCHAR,
    display_name VARCHAR,
    avatar_url TEXT,
    is_verified BOOLEAN,
    online_status VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.user_id,
        p.username,
        p.display_name,
        p.avatar_url,
        p.is_verified,
        p.online_status
    FROM users.profiles p
    INNER JOIN auth.users u ON u.id = p.user_id
    WHERE 
        p.search_visibility = TRUE
        AND u.account_status = 'active'
        AND u.deleted_at IS NULL
        AND p.deactivated_at IS NULL
        AND (
            p.username ILIKE '%' || search_query || '%'
            OR p.display_name ILIKE '%' || search_query || '%'
            OR u.email ILIKE '%' || search_query || '%'
        )
    ORDER BY
        CASE WHEN p.is_verified THEN 0 ELSE 1 END,
        p.username
    LIMIT limit_count
    OFFSET offset_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- Get user's active conversations with pagination
CREATE OR REPLACE FUNCTION messages.get_user_conversations(
    p_user_id UUID,
    limit_count INTEGER DEFAULT 20,
    offset_count INTEGER DEFAULT 0
)
RETURNS TABLE (
    conversation_id UUID,
    conversation_type VARCHAR,
    title VARCHAR,
    avatar_url TEXT,
    last_message_at TIMESTAMPTZ,
    unread_count INTEGER,
    is_pinned BOOLEAN,
    is_muted BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        c.id,
        c.conversation_type,
        c.title,
        c.avatar_url,
        c.last_message_at,
        p.unread_count,
        p.is_pinned,
        p.is_muted
    FROM messages.conversations c
    INNER JOIN messages.conversation_participants p ON p.conversation_id = c.id
    WHERE 
        p.user_id = p_user_id
        AND p.left_at IS NULL
        AND p.removed_at IS NULL
        AND c.is_active = TRUE
    ORDER BY 
        p.is_pinned DESC,
        c.last_activity_at DESC
    LIMIT limit_count
    OFFSET offset_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- Batch mark messages as read
CREATE OR REPLACE FUNCTION messages.mark_messages_as_read(
    p_user_id UUID,
    p_conversation_id UUID,
    p_last_message_id UUID
)
RETURNS VOID AS $$
BEGIN
    UPDATE messages.conversation_participants
    SET 
        last_read_message_id = p_last_message_id,
        last_read_at = NOW(),
        unread_count = 0
    WHERE user_id = p_user_id
    AND conversation_id = p_conversation_id;
    
    -- Also update message delivery status
    UPDATE messages.message_delivery_status
    SET 
        status = 'read',
        read_at = NOW()
    WHERE user_id = p_user_id
    AND message_id IN (
        SELECT id FROM messages.messages
        WHERE conversation_id = p_conversation_id
        AND created_at <= (
            SELECT created_at FROM messages.messages WHERE id = p_last_message_id
        )
    )
    AND status != 'read';
END;
$$ LANGUAGE plpgsql;

-- Get user statistics
CREATE OR REPLACE FUNCTION users.get_user_stats(p_user_id UUID)
RETURNS JSON AS $$
DECLARE
    stats JSON;
BEGIN
    SELECT json_build_object(
        'total_contacts', (
            SELECT COUNT(*) FROM users.contacts
            WHERE user_id = p_user_id AND status = 'accepted'
        ),
        'total_conversations', (
            SELECT COUNT(*) FROM messages.conversation_participants
            WHERE user_id = p_user_id AND left_at IS NULL AND removed_at IS NULL
        ),
        'total_messages_sent', (
            SELECT COUNT(*) FROM messages.messages
            WHERE sender_user_id = p_user_id AND deleted_at IS NULL
        ),
        'unread_messages', (
            SELECT COALESCE(SUM(unread_count), 0)
            FROM messages.conversation_participants
            WHERE user_id = p_user_id AND left_at IS NULL AND removed_at IS NULL
        ),
        'storage_used', (
            SELECT COALESCE(SUM(file_size), 0)
            FROM media.media_files
            WHERE uploaded_by_user_id = p_user_id
        ),
        'account_created_at', (
            SELECT created_at FROM auth.users WHERE id = p_user_id
        ),
        'last_seen_at', (
            SELECT last_seen_at FROM users.profiles WHERE user_id = p_user_id
        )
    ) INTO stats;
    
    RETURN stats;
END;
$$ LANGUAGE plpgsql STABLE;

-- Cleanup old data
CREATE OR REPLACE FUNCTION public.cleanup_old_data()
RETURNS JSON AS $$
DECLARE
    result JSON;
    sessions_deleted INTEGER;
    otps_deleted INTEGER;
    notifications_deleted INTEGER;
BEGIN
    -- Delete expired sessions
    SELECT auth.clean_expired_sessions() INTO sessions_deleted;
    
    -- Delete expired OTPs
    SELECT auth.clean_expired_otps() INTO otps_deleted;
    
    -- Delete old read notifications (>30 days)
    WITH deleted AS (
        DELETE FROM notifications.notifications
        WHERE read_at IS NOT NULL
        AND read_at < NOW() - INTERVAL '30 days'
        RETURNING id
    )
    SELECT COUNT(*) INTO notifications_deleted FROM deleted;
    
    -- Build result
    SELECT json_build_object(
        'sessions_deleted', sessions_deleted,
        'otps_deleted', otps_deleted,
        'notifications_deleted', notifications_deleted,
        'cleaned_at', NOW()
    ) INTO result;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Paginate query results with cursor
CREATE OR REPLACE FUNCTION public.cursor_paginate(
    query_text TEXT,
    cursor_value TEXT DEFAULT NULL,
    page_size INTEGER DEFAULT 20
)
RETURNS JSON AS $$
DECLARE
    result JSON;
BEGIN
    -- This is a simplified version - actual implementation would be more complex
    EXECUTE format('
        SELECT json_build_object(
            ''data'', json_agg(row_to_json(t)),
            ''cursor'', MAX(t.id)
        )
        FROM (%s) t
        WHERE ($1 IS NULL OR t.id > $1::UUID)
        LIMIT $2
    ', query_text)
    INTO result
    USING cursor_value, page_size;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Analyze table and return statistics
CREATE OR REPLACE FUNCTION public.get_table_stats(schema_name TEXT, table_name TEXT)
RETURNS JSON AS $$
DECLARE
    stats JSON;
BEGIN
    SELECT json_build_object(
        'row_count', (
            SELECT reltuples::BIGINT
            FROM pg_class
            WHERE oid = (schema_name || '.' || table_name)::regclass
        ),
        'table_size', (
            SELECT pg_size_pretty(pg_total_relation_size((schema_name || '.' || table_name)::regclass))
        ),
        'index_size', (
            SELECT pg_size_pretty(pg_indexes_size((schema_name || '.' || table_name)::regclass))
        ),
        'last_vacuum', (
            SELECT last_vacuum FROM pg_stat_user_tables
            WHERE schemaname = schema_name AND relname = table_name
        ),
        'last_analyze', (
            SELECT last_analyze FROM pg_stat_user_tables
            WHERE schemaname = schema_name AND relname = table_name
        )
    ) INTO stats;
    
    RETURN stats;
END;
$$ LANGUAGE plpgsql;
