-- =====================================================
-- RLS (Row Level Security) Functions
-- =====================================================

-- Function to check if user is authenticated
CREATE OR REPLACE FUNCTION auth.current_user_id()
RETURNS UUID AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_user_id', TRUE), '')::UUID;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to check if user is admin
CREATE OR REPLACE FUNCTION auth.is_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM auth.users
        WHERE id = auth.current_user_id()
        AND account_status = 'active'
        AND EXISTS (
            SELECT 1 FROM users.profiles
            WHERE user_id = auth.current_user_id()
            AND metadata->>'role' = 'admin'
        )
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to check if user is conversation participant
CREATE OR REPLACE FUNCTION messages.is_conversation_participant(conv_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = conv_id
        AND user_id = auth.current_user_id()
        AND left_at IS NULL
        AND removed_at IS NULL
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to check if user can view profile
CREATE OR REPLACE FUNCTION users.can_view_profile(target_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    profile_visibility VARCHAR(20);
    is_contact BOOLEAN;
BEGIN
    -- Get profile visibility
    SELECT p.profile_visibility INTO profile_visibility
    FROM users.profiles p
    WHERE p.user_id = target_user_id;

    -- Public profiles are visible to everyone
    IF profile_visibility = 'public' THEN
        RETURN TRUE;
    END IF;

    -- Private profiles only visible to owner
    IF profile_visibility = 'private' THEN
        RETURN target_user_id = auth.current_user_id();
    END IF;

    -- Friends/contacts only
    IF profile_visibility = 'friends' THEN
        -- Check if users are contacts
        SELECT EXISTS (
            SELECT 1 FROM users.contacts
            WHERE ((user_id = auth.current_user_id() AND contact_user_id = target_user_id)
               OR (user_id = target_user_id AND contact_user_id = auth.current_user_id()))
            AND status = 'accepted'
        ) INTO is_contact;
        
        RETURN target_user_id = auth.current_user_id() OR is_contact;
    END IF;

    RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to check if users are contacts
CREATE OR REPLACE FUNCTION users.are_contacts(user1_id UUID, user2_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users.contacts
        WHERE ((user_id = user1_id AND contact_user_id = user2_id)
           OR (user_id = user2_id AND contact_user_id = user1_id))
        AND status = 'accepted'
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to check if user has blocked another user
CREATE OR REPLACE FUNCTION users.has_blocked(blocker_id UUID, blocked_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users.contacts
        WHERE user_id = blocker_id
        AND contact_user_id = blocked_id
        AND status = 'blocked'
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to get user's unread message count
CREATE OR REPLACE FUNCTION messages.get_unread_count(p_user_id UUID)
RETURNS BIGINT AS $$
BEGIN
    RETURN (
        SELECT COALESCE(SUM(unread_count), 0)
        FROM messages.conversation_participants
        WHERE user_id = p_user_id
        AND left_at IS NULL
        AND removed_at IS NULL
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to check conversation permissions
CREATE OR REPLACE FUNCTION messages.has_conversation_permission(
    conv_id UUID,
    permission_type VARCHAR(50)
)
RETURNS BOOLEAN AS $$
DECLARE
    has_perm BOOLEAN;
BEGIN
    SELECT 
        CASE permission_type
            WHEN 'send_messages' THEN can_send_messages
            WHEN 'send_media' THEN can_send_media
            WHEN 'add_members' THEN can_add_members
            WHEN 'remove_members' THEN can_remove_members
            WHEN 'edit_info' THEN can_edit_info
            WHEN 'pin_messages' THEN can_pin_messages
            WHEN 'delete_messages' THEN can_delete_messages
            ELSE FALSE
        END INTO has_perm
    FROM messages.conversation_participants
    WHERE conversation_id = conv_id
    AND user_id = auth.current_user_id()
    AND left_at IS NULL
    AND removed_at IS NULL;

    RETURN COALESCE(has_perm, FALSE);
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to check if user owns media
CREATE OR REPLACE FUNCTION media.owns_media(media_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM media.media_files
        WHERE id = media_id
        AND uploaded_by_user_id = auth.current_user_id()
    );
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to sanitize and validate email
CREATE OR REPLACE FUNCTION auth.validate_email(email_input TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN email_input ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to check password strength
CREATE OR REPLACE FUNCTION auth.is_strong_password(password_input TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    -- At least 8 characters, contains uppercase, lowercase, number, and special character
    RETURN LENGTH(password_input) >= 8
        AND password_input ~ '[A-Z]'
        AND password_input ~ '[a-z]'
        AND password_input ~ '[0-9]'
        AND password_input ~ '[^A-Za-z0-9]';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to increment failed login attempts
CREATE OR REPLACE FUNCTION auth.increment_failed_login_attempts(p_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE auth.users
    SET 
        failed_login_attempts = failed_login_attempts + 1,
        last_failed_login_at = NOW(),
        account_locked_until = CASE
            WHEN failed_login_attempts + 1 >= 5 THEN NOW() + INTERVAL '30 minutes'
            ELSE account_locked_until
        END
    WHERE id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Function to reset failed login attempts
CREATE OR REPLACE FUNCTION auth.reset_failed_login_attempts(p_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE auth.users
    SET 
        failed_login_attempts = 0,
        last_failed_login_at = NULL,
        last_successful_login_at = NOW(),
        account_locked_until = NULL
    WHERE id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Function to clean expired sessions
CREATE OR REPLACE FUNCTION auth.clean_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    WITH deleted AS (
        DELETE FROM auth.sessions
        WHERE expires_at < NOW()
        AND revoked_at IS NULL
        RETURNING id
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean expired OTPs
CREATE OR REPLACE FUNCTION auth.clean_expired_otps()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    WITH deleted AS (
        DELETE FROM auth.otp_verifications
        WHERE expires_at < NOW()
        OR (is_verified = TRUE AND verified_at < NOW() - INTERVAL '24 hours')
        RETURNING id
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
