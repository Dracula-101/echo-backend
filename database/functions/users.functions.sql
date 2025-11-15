-- =====================================================
-- USERS SCHEMA - FUNCTIONS
-- =====================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION users.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update contact interaction tracking
CREATE OR REPLACE FUNCTION users.update_contact_interaction()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users.contacts
    SET last_interaction_at = NOW(),
        interaction_count = interaction_count + 1
    WHERE user_id = NEW.user_id
    AND contact_user_id = NEW.contact_user_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to check if users are contacts
CREATE OR REPLACE FUNCTION users.are_contacts(p_user_id UUID, p_other_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users.contacts
        WHERE user_id = p_user_id
        AND contact_user_id = p_other_user_id
        AND status = 'accepted'
    );
END;
$$ LANGUAGE plpgsql;

-- Function to check if user is blocked
CREATE OR REPLACE FUNCTION users.is_blocked(p_user_id UUID, p_other_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users.blocked_users
        WHERE user_id = p_user_id
        AND blocked_user_id = p_other_user_id
        AND unblocked_at IS NULL
    );
END;
$$ LANGUAGE plpgsql;

-- Function to block user
CREATE OR REPLACE FUNCTION users.block_user(
    p_user_id UUID,
    p_blocked_user_id UUID,
    p_block_reason TEXT DEFAULT NULL,
    p_block_type VARCHAR DEFAULT 'full'
)
RETURNS UUID AS $$
DECLARE
    v_block_id UUID;
BEGIN
    -- Check if already blocked
    IF users.is_blocked(p_user_id, p_blocked_user_id) THEN
        RAISE EXCEPTION 'User is already blocked';
    END IF;
    
    -- Insert block record
    INSERT INTO users.blocked_users (
        user_id, blocked_user_id, block_reason, block_type
    ) VALUES (
        p_user_id, p_blocked_user_id, p_block_reason, p_block_type
    ) RETURNING id INTO v_block_id;
    
    -- Update contact status if exists
    UPDATE users.contacts
    SET relationship_type = 'blocked',
        status = 'blocked',
        blocked_at = NOW(),
        block_reason = p_block_reason
    WHERE user_id = p_user_id
    AND contact_user_id = p_blocked_user_id;
    
    RETURN v_block_id;
END;
$$ LANGUAGE plpgsql;

-- Function to unblock user
CREATE OR REPLACE FUNCTION users.unblock_user(p_user_id UUID, p_blocked_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE users.blocked_users
    SET unblocked_at = NOW()
    WHERE user_id = p_user_id
    AND blocked_user_id = p_blocked_user_id
    AND unblocked_at IS NULL;
    
    -- Optionally restore contact status
    UPDATE users.contacts
    SET relationship_type = 'contact',
        status = 'accepted'
    WHERE user_id = p_user_id
    AND contact_user_id = p_blocked_user_id
    AND relationship_type = 'blocked';
END;
$$ LANGUAGE plpgsql;

-- Function to update online status
CREATE OR REPLACE FUNCTION users.update_online_status(
    p_user_id UUID,
    p_status VARCHAR,
    p_last_seen TIMESTAMPTZ DEFAULT NOW()
)
RETURNS VOID AS $$
BEGIN
    UPDATE users.profiles
    SET online_status = p_status,
        last_seen_at = p_last_seen,
        updated_at = NOW()
    WHERE user_id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Function to set user offline after inactivity
CREATE OR REPLACE FUNCTION users.set_inactive_users_offline(p_minutes INTEGER DEFAULT 5)
RETURNS INTEGER AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    UPDATE users.profiles
    SET online_status = 'offline'
    WHERE online_status IN ('online', 'away')
    AND last_seen_at < NOW() - (p_minutes || ' minutes')::INTERVAL;
    
    GET DIAGNOSTICS v_updated_count = ROW_COUNT;
    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean expired statuses
CREATE OR REPLACE FUNCTION users.cleanup_expired_statuses()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    UPDATE users.status_history
    SET deleted_at = NOW()
    WHERE expires_at < NOW()
    AND deleted_at IS NULL;
    
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get user's contacts
CREATE OR REPLACE FUNCTION users.get_user_contacts(
    p_user_id UUID,
    p_relationship_type VARCHAR DEFAULT NULL
)
RETURNS TABLE (
    contact_id UUID,
    username VARCHAR,
    display_name VARCHAR,
    avatar_url TEXT,
    online_status VARCHAR,
    last_seen_at TIMESTAMPTZ,
    is_favorite BOOLEAN,
    is_pinned BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        c.contact_user_id,
        p.username,
        p.display_name,
        p.avatar_url,
        p.online_status,
        p.last_seen_at,
        c.is_favorite,
        c.is_pinned
    FROM users.contacts c
    JOIN users.profiles p ON p.user_id = c.contact_user_id
    WHERE c.user_id = p_user_id
    AND c.status = 'accepted'
    AND (p_relationship_type IS NULL OR c.relationship_type = p_relationship_type)
    ORDER BY c.is_pinned DESC, c.is_favorite DESC, p.display_name ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to search users
CREATE OR REPLACE FUNCTION users.search_users(
    p_search_term TEXT,
    p_limit INTEGER DEFAULT 20
)
RETURNS TABLE (
    user_id UUID,
    username VARCHAR,
    display_name VARCHAR,
    avatar_url TEXT,
    bio TEXT,
    is_verified BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.user_id,
        p.username,
        p.display_name,
        p.avatar_url,
        p.bio,
        p.is_verified
    FROM users.profiles p
    WHERE p.search_visibility = TRUE
    AND p.deactivated_at IS NULL
    AND (
        p.username ILIKE '%' || p_search_term || '%'
        OR p.display_name ILIKE '%' || p_search_term || '%'
        OR to_tsvector('english', COALESCE(p.display_name, '') || ' ' || COALESCE(p.username, '') || ' ' || COALESCE(p.bio, ''))
        @@ plainto_tsquery('english', p_search_term)
    )
    ORDER BY 
        CASE WHEN p.username = p_search_term THEN 0 ELSE 1 END,
        p.is_verified DESC,
        p.display_name
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to log user activity
CREATE OR REPLACE FUNCTION users.log_activity(
    p_user_id UUID,
    p_activity_type VARCHAR,
    p_activity_category VARCHAR,
    p_description TEXT,
    p_old_value JSONB DEFAULT NULL,
    p_new_value JSONB DEFAULT NULL,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_device_id VARCHAR DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_log_id UUID;
BEGIN
    INSERT INTO users.activity_log (
        user_id, activity_type, activity_category, description,
        old_value, new_value, ip_address, user_agent, device_id
    ) VALUES (
        p_user_id, p_activity_type, p_activity_category, p_description,
        p_old_value, p_new_value, p_ip_address, p_user_agent, p_device_id
    ) RETURNING id INTO v_log_id;
    
    RETURN v_log_id;
END;
$$ LANGUAGE plpgsql;

-- Function to increment status view count
CREATE OR REPLACE FUNCTION users.increment_status_views()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users.status_history
    SET views_count = views_count + 1
    WHERE id = NEW.status_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update contact group member count
CREATE OR REPLACE FUNCTION users.update_contact_group_count(p_group_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE users.contact_groups
    SET member_count = (
        SELECT COUNT(*)
        FROM users.contacts
        WHERE p_group_id = ANY(contact_groups)
        AND user_id = users.contact_groups.user_id
    )
    WHERE id = p_group_id;
END;
$$ LANGUAGE plpgsql;

-- Function to register device
CREATE OR REPLACE FUNCTION users.register_device(
    p_user_id UUID,
    p_device_id VARCHAR,
    p_device_name VARCHAR,
    p_device_type VARCHAR,
    p_device_model VARCHAR,
    p_device_manufacturer VARCHAR,
    p_os_name VARCHAR,
    p_os_version VARCHAR,
    p_app_version VARCHAR,
    p_fcm_token TEXT DEFAULT NULL,
    p_apns_token TEXT DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_device_record_id UUID;
BEGIN
    -- Set all other devices as not current
    UPDATE users.devices
    SET is_current_device = FALSE
    WHERE user_id = p_user_id;
    
    -- Insert or update device
    INSERT INTO users.devices (
        user_id, device_id, device_name, device_type, device_model,
        device_manufacturer, os_name, os_version, app_version,
        fcm_token, apns_token, is_current_device, last_active_at
    ) VALUES (
        p_user_id, p_device_id, p_device_name, p_device_type, p_device_model,
        p_device_manufacturer, p_os_name, p_os_version, p_app_version,
        p_fcm_token, p_apns_token, TRUE, NOW()
    )
    ON CONFLICT (user_id, device_id) DO UPDATE SET
        device_name = EXCLUDED.device_name,
        device_type = EXCLUDED.device_type,
        is_current_device = TRUE,
        last_active_at = NOW(),
        fcm_token = COALESCE(EXCLUDED.fcm_token, users.devices.fcm_token),
        apns_token = COALESCE(EXCLUDED.apns_token, users.devices.apns_token),
        is_active = TRUE
    RETURNING id INTO v_device_record_id;
    
    RETURN v_device_record_id;
END;
$$ LANGUAGE plpgsql;