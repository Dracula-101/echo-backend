-- =====================================================
-- PERFORMANCE INDEXES
-- =====================================================

-- Auth schema indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON auth.users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_phone ON auth.users(phone_number) WHERE phone_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_account_status ON auth.users(account_status) WHERE account_status != 'deleted';
CREATE INDEX IF NOT EXISTS idx_users_created_at ON auth.users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_email_verified ON auth.users(email_verified) WHERE email_verified = TRUE;

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON auth.sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_token ON auth.sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON auth.sessions(expires_at) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_sessions_device_id ON auth.sessions(device_id);
CREATE INDEX IF NOT EXISTS idx_sessions_active ON auth.sessions(user_id, is_active) WHERE is_active = TRUE;

-- Users schema indexes
CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON users.profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_profiles_username ON users.profiles(username) WHERE deactivated_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_profiles_search ON users.profiles USING gin(to_tsvector('english', COALESCE(username, '') || ' ' || COALESCE(display_name, '')));
CREATE INDEX IF NOT EXISTS idx_profiles_visibility ON users.profiles(profile_visibility, search_visibility);
CREATE INDEX IF NOT EXISTS idx_profiles_online_status ON users.profiles(online_status, last_seen_at);
CREATE INDEX IF NOT EXISTS idx_profiles_is_verified ON users.profiles(is_verified) WHERE is_verified = TRUE;

CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON users.contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_contact_user_id ON users.contacts(contact_user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_relationship ON users.contacts(user_id, relationship_type, status);
CREATE INDEX IF NOT EXISTS idx_contacts_status ON users.contacts(status) WHERE status = 'accepted';
CREATE INDEX IF NOT EXISTS idx_contacts_composite ON users.contacts(user_id, contact_user_id, status);

CREATE INDEX IF NOT EXISTS idx_blocks_blocker_id ON users.blocks(blocker_user_id);
CREATE INDEX IF NOT EXISTS idx_blocks_blocked_id ON users.blocks(blocked_user_id);
CREATE INDEX IF NOT EXISTS idx_blocks_composite ON users.blocks(blocker_user_id, blocked_user_id);

-- Messages schema indexes
CREATE INDEX IF NOT EXISTS idx_conversations_creator ON messages.conversations(creator_user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_type ON messages.conversations(conversation_type);
CREATE INDEX IF NOT EXISTS idx_conversations_last_activity ON messages.conversations(last_activity_at DESC) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_conversations_invite_link ON messages.conversations(invite_link) WHERE invite_link IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_conversations_public ON messages.conversations(is_public) WHERE is_public = TRUE;

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages.messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages.messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages.messages(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_created ON messages.messages(conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_deleted ON messages.messages(conversation_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_messages_parent_id ON messages.messages(parent_message_id) WHERE parent_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_search ON messages.messages USING gin(to_tsvector('english', content));

CREATE INDEX IF NOT EXISTS idx_participants_conversation_id ON messages.participants(conversation_id);
CREATE INDEX IF NOT EXISTS idx_participants_user_id ON messages.participants(user_id);
CREATE INDEX IF NOT EXISTS idx_participants_composite ON messages.participants(conversation_id, user_id);
CREATE INDEX IF NOT EXISTS idx_participants_active ON messages.participants(user_id, status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_participants_role ON messages.participants(conversation_id, role);

CREATE INDEX IF NOT EXISTS idx_reactions_message_id ON messages.reactions(message_id);
CREATE INDEX IF NOT EXISTS idx_reactions_user_id ON messages.reactions(user_id);
CREATE INDEX IF NOT EXISTS idx_reactions_composite ON messages.reactions(message_id, user_id);

CREATE INDEX IF NOT EXISTS idx_read_receipts_message_id ON messages.read_receipts(message_id);
CREATE INDEX IF NOT EXISTS idx_read_receipts_user_id ON messages.read_receipts(user_id);
CREATE INDEX IF NOT EXISTS idx_read_receipts_composite ON messages.read_receipts(message_id, user_id);

-- Media schema indexes
CREATE INDEX IF NOT EXISTS idx_media_files_uploaded_by ON media.files(uploaded_by_user_id);
CREATE INDEX IF NOT EXISTS idx_media_files_type ON media.files(file_type);
CREATE INDEX IF NOT EXISTS idx_media_files_created_at ON media.files(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_media_files_conversation ON media.files(conversation_id) WHERE conversation_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_media_files_message ON media.files(message_id) WHERE message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_media_files_status ON media.files(processing_status);

CREATE INDEX IF NOT EXISTS idx_thumbnails_media_id ON media.thumbnails(media_id);
CREATE INDEX IF NOT EXISTS idx_thumbnails_size ON media.thumbnails(size_variant);

-- Notifications schema indexes
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications.notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications.notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications.notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications.notifications(notification_type);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications.notifications(user_id, created_at DESC) WHERE is_read = FALSE;

CREATE INDEX IF NOT EXISTS idx_push_tokens_user_id ON notifications.push_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_push_tokens_token ON notifications.push_tokens(token);
CREATE INDEX IF NOT EXISTS idx_push_tokens_active ON notifications.push_tokens(user_id, is_active) WHERE is_active = TRUE;

-- Analytics schema indexes
CREATE INDEX IF NOT EXISTS idx_analytics_events_user_id ON analytics.events(user_id);
CREATE INDEX IF NOT EXISTS idx_analytics_events_type ON analytics.events(event_type);
CREATE INDEX IF NOT EXISTS idx_analytics_events_created_at ON analytics.events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_events_composite ON analytics.events(user_id, event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_events_session ON analytics.events(session_id) WHERE session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_analytics_user_sessions_user_id ON analytics.user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_analytics_user_sessions_started ON analytics.user_sessions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_user_sessions_duration ON analytics.user_sessions(session_duration);

-- Location schema indexes
CREATE INDEX IF NOT EXISTS idx_ip_locations_ip ON location.ip_locations(ip_address);
CREATE INDEX IF NOT EXISTS idx_ip_locations_country ON location.ip_locations(country_code);
CREATE INDEX IF NOT EXISTS idx_ip_locations_city ON location.ip_locations(city);

CREATE INDEX IF NOT EXISTS idx_user_locations_user_id ON location.user_locations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_locations_created_at ON location.user_locations(created_at DESC);

-- Partial indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_users_active_verified ON auth.users(id) 
    WHERE account_status = 'active' AND email_verified = TRUE;

CREATE INDEX IF NOT EXISTS idx_conversations_active_recent ON messages.conversations(last_activity_at DESC) 
    WHERE is_active = TRUE AND is_archived = FALSE;

CREATE INDEX IF NOT EXISTS idx_messages_recent_undeleted ON messages.messages(conversation_id, created_at DESC) 
    WHERE deleted_at IS NULL;

-- Covering indexes for common queries
CREATE INDEX IF NOT EXISTS idx_participants_with_last_read ON messages.participants(conversation_id, user_id) 
    INCLUDE (last_read_at, role, status);

CREATE INDEX IF NOT EXISTS idx_profiles_public_search ON users.profiles(username) 
    INCLUDE (display_name, avatar_url, is_verified)
    WHERE deactivated_at IS NULL AND profile_visibility = 'public';

-- B-tree indexes for sorting and range queries
CREATE INDEX IF NOT EXISTS idx_messages_created_at_btree ON messages.messages USING btree(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_created_at_btree ON auth.users USING btree(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_events_timestamp ON analytics.events USING btree(created_at DESC);

-- Hash indexes for equality searches
CREATE INDEX IF NOT EXISTS idx_users_email_hash ON auth.users USING hash(email);
CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON auth.sessions USING hash(session_token);

-- Comment indexes with their purpose
COMMENT ON INDEX idx_messages_search IS 'Full-text search index for message content';
COMMENT ON INDEX idx_profiles_search IS 'Full-text search index for user profiles';
COMMENT ON INDEX idx_messages_conversation_created IS 'Composite index for conversation timeline queries';
COMMENT ON INDEX idx_participants_with_last_read IS 'Covering index for unread message calculations';
