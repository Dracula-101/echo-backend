-- =====================================================
-- Trigger Functions for Automated Operations
-- =====================================================

-- Update updated_at timestamp
CREATE OR REPLACE FUNCTION public.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Update conversation last_message info
CREATE OR REPLACE FUNCTION messages.update_conversation_last_message()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE messages.conversations
        SET 
            last_message_id = NEW.id,
            last_message_at = NEW.created_at,
            last_activity_at = NEW.created_at,
            message_count = message_count + 1
        WHERE id = NEW.conversation_id;
        
        -- Update unread counts for all participants except sender
        UPDATE messages.conversation_participants
        SET unread_count = unread_count + 1
        WHERE conversation_id = NEW.conversation_id
        AND user_id != NEW.sender_user_id
        AND left_at IS NULL
        AND removed_at IS NULL;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Update conversation message count on delete
CREATE OR REPLACE FUNCTION messages.update_conversation_message_count_on_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversations
    SET message_count = GREATEST(message_count - 1, 0)
    WHERE id = OLD.conversation_id;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Update participant unread count when message is read
CREATE OR REPLACE FUNCTION messages.update_participant_read_status()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.last_read_message_id IS DISTINCT FROM OLD.last_read_message_id THEN
        NEW.last_read_at = NOW();
        NEW.unread_count = 0;
        NEW.mention_count = 0;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Update conversation member count
CREATE OR REPLACE FUNCTION messages.update_conversation_member_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE messages.conversations
        SET member_count = member_count + 1
        WHERE id = NEW.conversation_id;
    ELSIF TG_OP = 'UPDATE' THEN
        IF NEW.left_at IS NOT NULL OR NEW.removed_at IS NOT NULL THEN
            IF OLD.left_at IS NULL AND OLD.removed_at IS NULL THEN
                UPDATE messages.conversations
                SET member_count = GREATEST(member_count - 1, 0)
                WHERE id = NEW.conversation_id;
            END IF;
        END IF;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE messages.conversations
        SET member_count = GREATEST(member_count - 1, 0)
        WHERE id = OLD.conversation_id;
    END IF;
    
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Update contact interaction stats
CREATE OR REPLACE FUNCTION users.update_contact_interaction()
RETURNS TRIGGER AS $$
BEGIN
    -- Update both user's contact records
    UPDATE users.contacts
    SET 
        last_interaction_at = NOW(),
        interaction_count = interaction_count + 1
    WHERE (user_id = NEW.sender_user_id AND contact_user_id IN (
        SELECT user_id FROM messages.conversation_participants
        WHERE conversation_id = NEW.conversation_id
        AND user_id != NEW.sender_user_id
    ))
    OR (contact_user_id = NEW.sender_user_id AND user_id IN (
        SELECT user_id FROM messages.conversation_participants
        WHERE conversation_id = NEW.conversation_id
        AND user_id != NEW.sender_user_id
    ));
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Update user online status
CREATE OR REPLACE FUNCTION users.update_last_seen()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        UPDATE users.profiles
        SET 
            last_seen_at = NOW(),
            online_status = 'online'
        WHERE user_id = NEW.user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Soft delete user account
CREATE OR REPLACE FUNCTION auth.soft_delete_user()
RETURNS TRIGGER AS $$
BEGIN
    NEW.deleted_at = NOW();
    NEW.account_status = 'deleted';
    NEW.email = 'deleted_' || NEW.id || '@deleted.local';
    NEW.phone_number = NULL;
    
    -- Deactivate profile
    UPDATE users.profiles
    SET 
        deactivated_at = NOW(),
        online_status = 'offline'
    WHERE user_id = NEW.id;
    
    -- Revoke all sessions
    UPDATE auth.sessions
    SET revoked_at = NOW(), revoked_reason = 'Account deleted'
    WHERE user_id = NEW.id AND revoked_at IS NULL;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Validate username format
CREATE OR REPLACE FUNCTION users.validate_username()
RETURNS TRIGGER AS $$
BEGIN
    -- Username must be alphanumeric with underscores, 3-50 characters
    IF NOT (NEW.username ~ '^[a-zA-Z0-9_]{3,50}$') THEN
        RAISE EXCEPTION 'Invalid username format. Use 3-50 alphanumeric characters or underscores.';
    END IF;
    
    -- Convert to lowercase
    NEW.username = LOWER(NEW.username);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Validate and update notification counts
CREATE OR REPLACE FUNCTION notifications.update_unread_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Increment unread count
        UPDATE users.profiles
        SET metadata = jsonb_set(
            COALESCE(metadata, '{}'::jsonb),
            '{unread_notifications}',
            to_jsonb(COALESCE((metadata->>'unread_notifications')::int, 0) + 1)
        )
        WHERE user_id = NEW.user_id;
    ELSIF TG_OP = 'UPDATE' AND NEW.read_at IS NOT NULL AND OLD.read_at IS NULL THEN
        -- Decrement unread count when notification is read
        UPDATE users.profiles
        SET metadata = jsonb_set(
            COALESCE(metadata, '{}'::jsonb),
            '{unread_notifications}',
            to_jsonb(GREATEST(COALESCE((metadata->>'unread_notifications')::int, 1) - 1, 0))
        )
        WHERE user_id = NEW.user_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Update media file stats
CREATE OR REPLACE FUNCTION media.update_media_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Update user's total storage used
        UPDATE users.profiles
        SET metadata = jsonb_set(
            COALESCE(metadata, '{}'::jsonb),
            '{storage_used}',
            to_jsonb(COALESCE((metadata->>'storage_used')::bigint, 0) + NEW.file_size)
        )
        WHERE user_id = NEW.uploaded_by_user_id;
    ELSIF TG_OP = 'DELETE' THEN
        -- Decrease storage used
        UPDATE users.profiles
        SET metadata = jsonb_set(
            COALESCE(metadata, '{}'::jsonb),
            '{storage_used}',
            to_jsonb(GREATEST(COALESCE((metadata->>'storage_used')::bigint, OLD.file_size) - OLD.file_size, 0))
        )
        WHERE user_id = OLD.uploaded_by_user_id;
    END IF;
    
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Prevent message editing after time limit
CREATE OR REPLACE FUNCTION messages.check_message_edit_time()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.content != NEW.content THEN
        -- Allow editing within 15 minutes
        IF OLD.created_at < NOW() - INTERVAL '15 minutes' THEN
            RAISE EXCEPTION 'Messages can only be edited within 15 minutes of sending';
        END IF;
        
        -- Track edit history
        NEW.is_edited = TRUE;
        NEW.edited_at = NOW();
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Auto-archive old conversations
CREATE OR REPLACE FUNCTION messages.auto_archive_old_conversations()
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
BEGIN
    WITH archived AS (
        UPDATE messages.conversations
        SET 
            is_archived = TRUE,
            archived_at = NOW()
        WHERE is_archived = FALSE
        AND last_activity_at < NOW() - INTERVAL '90 days'
        AND is_group = FALSE
        RETURNING id
    )
    SELECT COUNT(*) INTO archived_count FROM archived;
    
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- Validate OTP attempts
CREATE OR REPLACE FUNCTION auth.validate_otp_attempts()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.attempts >= NEW.max_attempts THEN
        RAISE EXCEPTION 'Maximum OTP verification attempts exceeded';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Log session creation
CREATE OR REPLACE FUNCTION auth.log_session_creation()
RETURNS TRIGGER AS $$
BEGIN
    -- Could insert into an audit log table
    PERFORM pg_notify(
        'session_created',
        json_build_object(
            'user_id', NEW.user_id,
            'ip_address', NEW.ip_address,
            'device_type', NEW.device_type,
            'created_at', NEW.created_at
        )::text
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
