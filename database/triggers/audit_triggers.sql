-- =====================================================
-- AUDIT TRIGGERS
-- =====================================================

-- Apply updated_at trigger to all tables with updated_at column

-- Auth schema
CREATE TRIGGER update_auth_users_updated_at
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auth_sessions_updated_at
    BEFORE UPDATE ON auth.sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Users schema
CREATE TRIGGER update_users_profiles_updated_at
    BEFORE UPDATE ON users.profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_contacts_updated_at
    BEFORE UPDATE ON users.contacts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_blocked_users_updated_at
    BEFORE UPDATE ON users.blocked_users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Messages schema
CREATE TRIGGER update_messages_conversations_updated_at
    BEFORE UPDATE ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_messages_messages_updated_at
    BEFORE UPDATE ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_messages_conversation_participants_updated_at
    BEFORE UPDATE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply conversation activity triggers
CREATE TRIGGER trigger_update_conversation_activity
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_conversation_activity();

-- Apply conversation member count triggers
CREATE TRIGGER trigger_update_conversation_member_count_insert
    AFTER INSERT ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_conversation_member_count();

CREATE TRIGGER trigger_update_conversation_member_count_update
    AFTER UPDATE OF status ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_conversation_member_count();

CREATE TRIGGER trigger_update_conversation_member_count_delete
    AFTER DELETE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_conversation_member_count();

-- Email validation trigger
CREATE TRIGGER validate_user_email
    BEFORE INSERT OR UPDATE OF email ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION validate_email();

-- Password hash validation trigger
CREATE TRIGGER validate_password_hash
    BEFORE INSERT OR UPDATE OF password_hash ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.hash_sensitive_data();

-- Failed login attempts trigger
CREATE TRIGGER log_failed_login_attempt
    BEFORE UPDATE OF failed_login_attempts ON auth.users
    FOR EACH ROW
    WHEN (NEW.failed_login_attempts > OLD.failed_login_attempts)
    EXECUTE FUNCTION auth.log_failed_login();

-- Media storage stats triggers
CREATE TRIGGER update_media_storage_on_insert
    AFTER INSERT ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_storage_stats();

CREATE TRIGGER update_media_storage_on_delete
    AFTER DELETE ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_storage_stats();

-- Notifications schema
CREATE TRIGGER update_notifications_updated_at
    BEFORE UPDATE ON notifications.notifications
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Auto-archive old notifications
CREATE TRIGGER auto_archive_notifications
    BEFORE UPDATE OF is_read ON notifications.notifications
    FOR EACH ROW
    WHEN (NEW.is_read = TRUE)
    EXECUTE FUNCTION notifications.auto_archive_old_notifications();

-- Analytics schema
CREATE TRIGGER update_analytics_events_updated_at
    BEFORE UPDATE ON analytics.events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Media schema
CREATE TRIGGER update_media_files_updated_at
    BEFORE UPDATE ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
