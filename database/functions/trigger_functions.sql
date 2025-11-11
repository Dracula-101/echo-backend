-- =====================================================
-- PRODUCTION TRIGGERS - Data Integrity & Automation
-- =====================================================

-- =====================================================
-- AUTH SCHEMA TRIGGERS
-- =====================================================


-- 1. Update timestamp trigger
CREATE OR REPLACE FUNCTION auth.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_auth_users_updated_at ON auth.users;
CREATE TRIGGER trigger_auth_users_updated_at
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();


CREATE OR REPLACE FUNCTION auth.log_security_event()
RETURNS TRIGGER AS $$
BEGIN
    -- Password change
    IF TG_OP = 'UPDATE' AND OLD.password_hash IS DISTINCT FROM NEW.password_hash THEN
        INSERT INTO auth.security_events (user_id, event_type, event_category, severity, description)
        VALUES (NEW.id, 'password_change', 'account_management', 'info', 'User password was changed');
    END IF;
    
    -- 2FA enabled/disabled
    IF TG_OP = 'UPDATE' AND OLD.two_factor_enabled IS DISTINCT FROM NEW.two_factor_enabled THEN
        INSERT INTO auth.security_events (user_id, event_type, event_category, severity, description)
        VALUES (NEW.id, 
                CASE WHEN NEW.two_factor_enabled THEN '2fa_enabled' ELSE '2fa_disabled' END,
                'security', 
                CASE WHEN NEW.two_factor_enabled THEN 'info' ELSE 'warning' END,
                CASE WHEN NEW.two_factor_enabled THEN 'Two-factor authentication enabled' ELSE 'Two-factor authentication disabled' END);
    END IF;
    
    -- Account status change
    IF TG_OP = 'UPDATE' AND OLD.account_status IS DISTINCT FROM NEW.account_status THEN
        INSERT INTO auth.security_events (user_id, event_type, event_category, severity, description)
        VALUES (NEW.id, 'account_status_change', 'account_management', 'warning',
                'Account status changed from ' || OLD.account_status || ' to ' || NEW.account_status);
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_auth_security_logging ON auth.users;
CREATE TRIGGER trigger_auth_security_logging
    AFTER UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.log_security_event();

CREATE OR REPLACE FUNCTION auth.revoke_expired_session()
RETURNS TRIGGER AS $$
BEGIN
    -- Auto-revoke expired sessions on any update
    IF NEW.expires_at < NOW() AND (NEW.revoked_at IS NULL OR OLD.revoked_at IS NULL) THEN
        NEW.revoked_at = NOW();
        NEW.revoked_reason = COALESCE(NEW.revoked_reason, 'Expired');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_auth_session_expiry ON auth.sessions;
CREATE TRIGGER trigger_auth_session_expiry
    BEFORE UPDATE ON auth.sessions
    FOR EACH ROW
    EXECUTE FUNCTION auth.revoke_expired_session();

-- =====================================================
-- USERS SCHEMA TRIGGERS
-- =====================================================

-- Update timestamp trigger for users.profiles
CREATE OR REPLACE FUNCTION users.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_users_profiles_updated_at ON users.profiles;
CREATE TRIGGER trigger_users_profiles_updated_at
    BEFORE UPDATE ON users.profiles
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_users_contacts_updated_at ON users.contacts;
CREATE TRIGGER trigger_users_contacts_updated_at
    BEFORE UPDATE ON users.contacts
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Sync profile updates to activity log
CREATE OR REPLACE FUNCTION users.log_profile_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        -- Log username changes
        IF OLD.username != NEW.username THEN
            INSERT INTO users.activity_log (user_id, activity_type, activity_category, description, old_value, new_value)
            VALUES (NEW.user_id, 'username_change', 'profile', 'Username changed',
                    jsonb_build_object('username', OLD.username),
                    jsonb_build_object('username', NEW.username));
        END IF;
        
        -- Log display name changes
        IF OLD.display_name IS DISTINCT FROM NEW.display_name THEN
            INSERT INTO users.activity_log (user_id, activity_type, activity_category, description, old_value, new_value)
            VALUES (NEW.user_id, 'display_name_change', 'profile', 'Display name changed',
                    jsonb_build_object('display_name', OLD.display_name),
                    jsonb_build_object('display_name', NEW.display_name));
        END IF;
        
        -- Log avatar changes
        IF OLD.avatar_url IS DISTINCT FROM NEW.avatar_url THEN
            INSERT INTO users.activity_log (user_id, activity_type, activity_category, description)
            VALUES (NEW.user_id, 'avatar_change', 'profile', 'Profile avatar updated');
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_profile_activity_log
    AFTER UPDATE ON users.profiles
    FOR EACH ROW
    EXECUTE FUNCTION users.log_profile_changes();

-- Auto-accept mutual contact requests
CREATE OR REPLACE FUNCTION users.handle_mutual_contact_request()
RETURNS TRIGGER AS $$
DECLARE
    reverse_contact_exists BOOLEAN;
BEGIN
    IF NEW.status = 'pending' THEN
        -- Check if reverse contact request exists
        SELECT EXISTS(
            SELECT 1 FROM users.contacts
            WHERE user_id = NEW.contact_user_id
            AND contact_user_id = NEW.user_id
            AND status = 'pending'
        ) INTO reverse_contact_exists;
        
        -- If mutual request, auto-accept both
        IF reverse_contact_exists THEN
            UPDATE users.contacts
            SET status = 'accepted', accepted_at = NOW()
            WHERE user_id = NEW.contact_user_id
            AND contact_user_id = NEW.user_id;
            
            NEW.status = 'accepted';
            NEW.accepted_at = NOW();
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_mutual_contact_acceptance
    BEFORE INSERT ON users.contacts
    FOR EACH ROW
    EXECUTE FUNCTION users.handle_mutual_contact_request();

-- Update contact interaction tracking
CREATE OR REPLACE FUNCTION users.update_contact_interaction()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users.contacts
    SET last_interaction_at = NOW(),
        interaction_count = interaction_count + 1
    WHERE (user_id = NEW.sender_user_id AND contact_user_id IN (
        SELECT user_id FROM messages.conversation_participants WHERE conversation_id = NEW.conversation_id
    )) OR (contact_user_id = NEW.sender_user_id AND user_id IN (
        SELECT user_id FROM messages.conversation_participants WHERE conversation_id = NEW.conversation_id
    ));
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_contact_interaction_tracking
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION users.update_contact_interaction();

-- =====================================================
-- MESSAGES SCHEMA TRIGGERS
-- =====================================================

-- Update conversation stats on new message
CREATE OR REPLACE FUNCTION messages.update_conversation_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversations
    SET message_count = message_count + 1,
        last_message_id = NEW.id,
        last_message_at = NEW.created_at,
        last_activity_at = NOW()
    WHERE id = NEW.conversation_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_conversation_stats
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_conversation_stats();

-- Update unread counts for participants
CREATE OR REPLACE FUNCTION messages.update_unread_counts()
RETURNS TRIGGER AS $$
BEGIN
    -- Increment unread count for all participants except sender
    UPDATE messages.conversation_participants
    SET unread_count = unread_count + 1
    WHERE conversation_id = NEW.conversation_id
    AND user_id != NEW.sender_user_id;
    
    -- Check for mentions and update mention counts
    IF NEW.mentions IS NOT NULL AND jsonb_array_length(NEW.mentions) > 0 THEN
        UPDATE messages.conversation_participants cp
        SET mention_count = mention_count + 1
        WHERE cp.conversation_id = NEW.conversation_id
        AND cp.user_id = ANY(
            SELECT (mention->>'user_id')::UUID 
            FROM jsonb_array_elements(NEW.mentions) AS mention
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_unread_counts
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_unread_counts();

-- Reset unread count on message read
CREATE OR REPLACE FUNCTION messages.reset_unread_on_read()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.last_read_message_id IS DISTINCT FROM OLD.last_read_message_id THEN
        NEW.unread_count = 0;
        NEW.mention_count = 0;
        NEW.last_read_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_reset_unread
    BEFORE UPDATE ON messages.conversation_participants
    FOR EACH ROW
    WHEN (NEW.last_read_message_id IS DISTINCT FROM OLD.last_read_message_id)
    EXECUTE FUNCTION messages.reset_unread_on_read();

-- Update reaction count on message
CREATE OR REPLACE FUNCTION messages.update_reaction_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE messages.messages
        SET reaction_count = reaction_count + 1
        WHERE id = NEW.message_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE messages.messages
        SET reaction_count = GREATEST(0, reaction_count - 1)
        WHERE id = OLD.message_id;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_reaction_count
    AFTER INSERT OR DELETE ON messages.reactions
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_reaction_count();

-- Update reply count on message
CREATE OR REPLACE FUNCTION messages.update_reply_count()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.parent_message_id IS NOT NULL THEN
        UPDATE messages.messages
        SET reply_count = reply_count + 1,
            last_reply_at = NEW.created_at
        WHERE id = NEW.parent_message_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_reply_count
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.parent_message_id IS NOT NULL)
    EXECUTE FUNCTION messages.update_reply_count();

-- Create delivery status records for new messages
CREATE OR REPLACE FUNCTION messages.create_delivery_status()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.delivery_status (message_id, user_id, status)
    SELECT NEW.id, user_id, 'sent'
    FROM messages.conversation_participants
    WHERE conversation_id = NEW.conversation_id
    AND user_id != NEW.sender_user_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_create_delivery_status
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.create_delivery_status();

-- Update conversation member count
CREATE OR REPLACE FUNCTION messages.update_member_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE messages.conversations
        SET member_count = member_count + 1
        WHERE id = NEW.conversation_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE messages.conversations
        SET member_count = GREATEST(0, member_count - 1)
        WHERE id = OLD.conversation_id;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_member_count
    AFTER INSERT OR DELETE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_member_count();

-- =====================================================
-- MEDIA SCHEMA TRIGGERS
-- =====================================================

-- Update timestamp trigger for media.files
CREATE OR REPLACE FUNCTION media.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_media_files_updated_at ON media.files;
CREATE TRIGGER trigger_media_files_updated_at
    BEFORE UPDATE ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Update storage stats on file upload
CREATE OR REPLACE FUNCTION media.update_storage_stats()
RETURNS TRIGGER AS $$
DECLARE
    category VARCHAR(50);
BEGIN
    category := NEW.file_category;
    
    IF TG_OP = 'INSERT' THEN
        INSERT INTO media.storage_stats (user_id, total_files, total_size_bytes)
        VALUES (NEW.uploader_user_id, 1, NEW.file_size_bytes)
        ON CONFLICT (user_id) DO UPDATE
        SET total_files = media.storage_stats.total_files + 1,
            total_size_bytes = media.storage_stats.total_size_bytes + NEW.file_size_bytes,
            last_calculated_at = NOW();
        
        -- Update category-specific stats
        IF category = 'image' THEN
            UPDATE media.storage_stats
            SET images_count = images_count + 1,
                images_size_bytes = images_size_bytes + NEW.file_size_bytes
            WHERE user_id = NEW.uploader_user_id;
        ELSIF category = 'video' THEN
            UPDATE media.storage_stats
            SET videos_count = videos_count + 1,
                videos_size_bytes = videos_size_bytes + NEW.file_size_bytes
            WHERE user_id = NEW.uploader_user_id;
        ELSIF category = 'audio' THEN
            UPDATE media.storage_stats
            SET audio_count = audio_count + 1,
                audio_size_bytes = audio_size_bytes + NEW.file_size_bytes
            WHERE user_id = NEW.uploader_user_id;
        ELSIF category = 'document' THEN
            UPDATE media.storage_stats
            SET documents_count = documents_count + 1,
                documents_size_bytes = documents_size_bytes + NEW.file_size_bytes
            WHERE user_id = NEW.uploader_user_id;
        END IF;
        
        -- Update storage used percentage
        UPDATE media.storage_stats
        SET storage_used_percentage = (total_size_bytes::DECIMAL / storage_quota_bytes::DECIMAL * 100)
        WHERE user_id = NEW.uploader_user_id;
        
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE media.storage_stats
        SET total_files = GREATEST(0, total_files - 1),
            total_size_bytes = GREATEST(0, total_size_bytes - OLD.file_size_bytes),
            last_calculated_at = NOW()
        WHERE user_id = OLD.uploader_user_id;
        
        -- Update category-specific stats
        IF category = 'image' THEN
            UPDATE media.storage_stats
            SET images_count = GREATEST(0, images_count - 1),
                images_size_bytes = GREATEST(0, images_size_bytes - OLD.file_size_bytes)
            WHERE user_id = OLD.uploader_user_id;
        ELSIF category = 'video' THEN
            UPDATE media.storage_stats
            SET videos_count = GREATEST(0, videos_count - 1),
                videos_size_bytes = GREATEST(0, videos_size_bytes - OLD.file_size_bytes)
            WHERE user_id = OLD.uploader_user_id;
        ELSIF category = 'audio' THEN
            UPDATE media.storage_stats
            SET audio_count = GREATEST(0, audio_count - 1),
                audio_size_bytes = GREATEST(0, audio_size_bytes - OLD.file_size_bytes)
            WHERE user_id = OLD.uploader_user_id;
        ELSIF category = 'document' THEN
            UPDATE media.storage_stats
            SET documents_count = GREATEST(0, documents_count - 1),
                documents_size_bytes = GREATEST(0, documents_size_bytes - OLD.file_size_bytes)
            WHERE user_id = OLD.uploader_user_id;
        END IF;
        
        -- Update storage used percentage
        UPDATE media.storage_stats
        SET storage_used_percentage = (total_size_bytes::DECIMAL / storage_quota_bytes::DECIMAL * 100)
        WHERE user_id = OLD.uploader_user_id;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_media_storage_stats
    AFTER INSERT OR DELETE ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_storage_stats();

-- Update album file count
CREATE OR REPLACE FUNCTION media.update_album_file_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE media.albums
        SET file_count = file_count + 1,
            updated_at = NOW()
        WHERE id = NEW.album_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE media.albums
        SET file_count = GREATEST(0, file_count - 1),
            updated_at = NOW()
        WHERE id = OLD.album_id;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_media_album_file_count
    AFTER INSERT OR DELETE ON media.album_files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_album_file_count();

-- Update sticker usage count
CREATE OR REPLACE FUNCTION media.increment_sticker_usage()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.message_type = 'sticker' THEN
        UPDATE media.stickers
        SET usage_count = usage_count + 1
        WHERE file_id IN (
            SELECT media_id FROM messages.message_media WHERE message_id = NEW.id
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_media_sticker_usage
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.message_type = 'sticker')
    EXECUTE FUNCTION media.increment_sticker_usage();

-- =====================================================
-- NOTIFICATIONS SCHEMA TRIGGERS
-- =====================================================

-- Update notification stats on delivery
CREATE OR REPLACE FUNCTION notifications.update_user_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO notifications.user_stats (user_id, total_notifications_sent, last_notification_at)
        VALUES (NEW.user_id, 1, NOW())
        ON CONFLICT (user_id) DO UPDATE
        SET total_notifications_sent = notifications.user_stats.total_notifications_sent + 1,
            last_notification_at = NOW(),
            updated_at = NOW();
    END IF;
    
    IF TG_OP = 'UPDATE' THEN
        -- Update delivered count
        IF OLD.delivery_status != 'delivered' AND NEW.delivery_status = 'delivered' THEN
            UPDATE notifications.user_stats
            SET total_notifications_delivered = total_notifications_delivered + 1
            WHERE user_id = NEW.user_id;
        END IF;
        
        -- Update opened count
        IF OLD.is_read = FALSE AND NEW.is_read = TRUE THEN
            UPDATE notifications.user_stats
            SET total_notifications_opened = total_notifications_opened + 1,
                last_opened_notification_at = NOW()
            WHERE user_id = NEW.user_id;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_notifications_user_stats
    AFTER INSERT OR UPDATE ON notifications.notifications
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_user_stats();

-- =====================================================
-- ANALYTICS SCHEMA TRIGGERS
-- =====================================================

-- Update daily active users on event
CREATE OR REPLACE FUNCTION analytics.track_daily_active_user()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.user_id IS NOT NULL THEN
        INSERT INTO analytics.daily_active_users (date, user_id, sessions_count)
        VALUES (CURRENT_DATE, NEW.user_id, 1)
        ON CONFLICT (date, user_id) DO UPDATE
        SET sessions_count = analytics.daily_active_users.sessions_count + 1;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_analytics_track_dau
    AFTER INSERT ON analytics.events
    FOR EACH ROW
    WHEN (NEW.event_name = 'app_open')
    EXECUTE FUNCTION analytics.track_daily_active_user();

-- Update user LTV on revenue event
CREATE OR REPLACE FUNCTION analytics.update_user_ltv()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO analytics.user_ltv (user_id, total_revenue, total_transactions, last_calculated_at)
    VALUES (NEW.user_id, NEW.amount_usd, 1, NOW())
    ON CONFLICT (user_id) DO UPDATE
    SET total_revenue = analytics.user_ltv.total_revenue + NEW.amount_usd,
        total_transactions = analytics.user_ltv.total_transactions + 1,
        average_transaction_value = (analytics.user_ltv.total_revenue + NEW.amount_usd) / (analytics.user_ltv.total_transactions + 1),
        last_calculated_at = NOW(),
        updated_at = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_analytics_update_ltv
    AFTER INSERT ON analytics.revenue_events
    FOR EACH ROW
    WHEN (NEW.status = 'completed')
    EXECUTE FUNCTION analytics.update_user_ltv();

-- =====================================================
-- LOCATION SCHEMA TRIGGERS
-- =====================================================

-- Update IP tracking stats
CREATE OR REPLACE FUNCTION location.update_ip_tracking()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE location.ip_addresses
    SET last_seen_at = NOW(),
        lookup_count = lookup_count + 1,
        user_count = user_count + CASE WHEN NEW.user_id IS NOT NULL THEN 1 ELSE 0 END
    WHERE ip_address = NEW.ip_address;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_location_ip_tracking
    AFTER INSERT ON location.user_locations
    FOR EACH ROW
    EXECUTE FUNCTION location.update_ip_tracking();

-- Detect location changes
CREATE OR REPLACE FUNCTION location.detect_location_changes()
RETURNS TRIGGER AS $$
DECLARE
    prev_location RECORD;
BEGIN
    -- Get previous location
    SELECT * INTO prev_location
    FROM location.user_locations
    WHERE user_id = NEW.user_id
    ORDER BY captured_at DESC
    LIMIT 1 OFFSET 1;
    
    IF prev_location IS NOT NULL THEN
        -- Check if new country
        IF prev_location.country IS DISTINCT FROM NEW.country THEN
            NEW.is_new_country = TRUE;
            NEW.is_new_location = TRUE;
        END IF;
        
        -- Check if new city
        IF prev_location.city IS DISTINCT FROM NEW.city THEN
            NEW.is_new_city = TRUE;
            NEW.is_new_location = TRUE;
        END IF;
    ELSE
        -- First location record
        NEW.is_new_location = TRUE;
        NEW.is_new_country = TRUE;
        NEW.is_new_city = TRUE;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_location_detect_changes
    BEFORE INSERT ON location.user_locations
    FOR EACH ROW
    EXECUTE FUNCTION location.detect_location_changes();

-- =====================================================
-- UTILITY FUNCTIONS
-- =====================================================

-- Function to clean up old records (call via scheduled job)
CREATE OR REPLACE FUNCTION cleanup_old_records()
RETURNS void AS $$
BEGIN
    -- Clean up expired OTP codes
    DELETE FROM auth.otp_verifications
    WHERE expires_at < NOW() - INTERVAL '7 days';
    
    -- Clean up old typing indicators
    DELETE FROM messages.typing_indicators
    WHERE expires_at < NOW();
    
    -- Clean up expired password reset tokens
    DELETE FROM auth.password_reset_tokens
    WHERE expires_at < NOW() - INTERVAL '7 days';
    
    -- Clean up old rate limit entries
    DELETE FROM auth.rate_limits
    WHERE window_start < NOW() - INTERVAL '24 hours';
    
    -- Clean up expired conversation invites
    DELETE FROM messages.conversation_invites
    WHERE expires_at < NOW() AND status = 'pending';
    
    -- Clean up expired location shares
    UPDATE location.location_shares
    SET is_active = FALSE, stopped_at = NOW()
    WHERE expires_at < NOW() AND is_active = TRUE;
END;
$$ LANGUAGE plpgsql;