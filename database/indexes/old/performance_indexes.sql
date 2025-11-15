-- -- =====================================================
-- -- PERFORMANCE INDEXES - Essential Query Optimization
-- -- =====================================================

-- -- =====================================================
-- -- AUTH SCHEMA INDEXES
-- -- =====================================================

-- -- Users - Login and authentication
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_active 
--     ON auth.users(email) WHERE deleted_at IS NULL;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_phone 
--     ON auth.users(phone_number) WHERE phone_verified = TRUE;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_status 
--     ON auth.users(account_status, created_at DESC);

-- -- Sessions - Active session lookups
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_user_active 
--     ON auth.sessions(user_id, last_activity_at DESC) 
--     WHERE revoked_at IS NULL;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_token 
--     ON auth.sessions(session_token) 
--     WHERE revoked_at IS NULL;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_expires 
--     ON auth.sessions(expires_at) WHERE revoked_at IS NULL;

-- -- OTP - Verification lookups
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_otp_identifier 
--     ON auth.otp_verifications(identifier, identifier_type, expires_at DESC);

-- -- Security events - Recent activity
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_security_events_user 
--     ON auth.security_events(user_id, created_at DESC);

-- -- Login history - Failed login tracking
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_login_history_user 
--     ON auth.login_history(user_id, created_at DESC);

-- -- API keys - Key validation
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_hash 
--     ON auth.api_keys(key_hash) 
--     WHERE is_active = TRUE;

-- -- =====================================================
-- -- USERS SCHEMA INDEXES
-- -- =====================================================

-- -- Profiles - Username and search
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_username 
--     ON users.profiles(LOWER(username));

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_search 
--     ON users.profiles USING gin(
--         to_tsvector('english', 
--             COALESCE(display_name, '') || ' ' || 
--             COALESCE(username, '') || ' ' || 
--             COALESCE(bio, '')
--         )
--     );

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_user 
--     ON users.profiles(user_id);

-- -- Contacts - Relationship lookups
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contacts_user_status 
--     ON users.contacts(user_id, status);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contacts_accepted 
--     ON users.contacts(user_id, contact_user_id) 
--     WHERE status = 'accepted';

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contacts_mutual 
--     ON users.contacts(contact_user_id, user_id, status);

-- -- Blocked users - Block checking
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blocked_users_check 
--     ON users.blocked_users(user_id, blocked_user_id) 
--     WHERE unblocked_at IS NULL;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blocked_users_reverse 
--     ON users.blocked_users(blocked_user_id, user_id) 
--     WHERE unblocked_at IS NULL;

-- -- Status history - Active status
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_status_history_active 
--     ON users.status_history(user_id, expires_at DESC) 
--     WHERE deleted_at IS NULL;

-- -- Devices - Push notifications
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_devices_user_active 
--     ON users.devices(user_id, is_active);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_devices_push 
--     ON users.devices(fcm_token) 
--     WHERE is_active = TRUE AND push_enabled = TRUE;

-- -- =====================================================
-- -- MESSAGES SCHEMA INDEXES
-- -- =====================================================

-- -- Conversations - User's conversation list
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_active 
--     ON messages.conversations(is_active, last_activity_at DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_type 
--     ON messages.conversations(conversation_type, is_active);

-- -- Participants - Conversation membership
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_participants_user 
--     ON messages.conversation_participants(user_id, conversation_id) 
--     WHERE left_at IS NULL AND removed_at IS NULL;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_participants_conversation 
--     ON messages.conversation_participants(conversation_id, user_id) 
--     WHERE left_at IS NULL;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_participants_unread 
--     ON messages.conversation_participants(user_id, unread_count) 
--     WHERE unread_count > 0;

-- -- Messages - Conversation history
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_conversation 
--     ON messages.messages(conversation_id, created_at DESC) 
--     WHERE is_deleted = FALSE;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_sender 
--     ON messages.messages(sender_user_id, created_at DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_parent 
--     ON messages.messages(parent_message_id, created_at DESC) 
--     WHERE parent_message_id IS NOT NULL;

-- -- Full-text search
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_search_content 
--     ON messages.search_index USING gin(content_tsvector);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_search_conversation 
--     ON messages.search_index(conversation_id, message_id);

-- -- Reactions - Message reactions
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reactions_message 
--     ON messages.reactions(message_id, user_id);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reactions_user 
--     ON messages.reactions(user_id, created_at DESC);

-- -- Delivery status - Message tracking
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_delivery_message_user 
--     ON messages.delivery_status(message_id, user_id, status);

-- -- Bookmarks - Saved messages
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_bookmarks_user 
--     ON messages.bookmarks(user_id, bookmarked_at DESC);

-- -- Calls - Call history
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calls_conversation 
--     ON messages.calls(conversation_id, created_at DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calls_user 
--     ON messages.calls(initiator_user_id, created_at DESC);

-- -- =====================================================
-- -- MEDIA SCHEMA INDEXES
-- -- =====================================================

-- -- Files - User's files
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_files_uploader 
--     ON media.files(uploader_user_id, created_at DESC) 
--     WHERE deleted_at IS NULL;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_files_hash 
--     ON media.files(content_hash) 
--     WHERE content_hash IS NOT NULL;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_files_processing 
--     ON media.files(processing_status, created_at) 
--     WHERE processing_status IN ('pending', 'processing');

-- -- Processing queue
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_processing_queue 
--     ON media.processing_queue(status, priority DESC, created_at);

-- -- Albums
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_albums_user 
--     ON media.albums(user_id, updated_at DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_album_files 
--     ON media.album_files(album_id, display_order);

-- -- Shares - Shared files
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shares_file 
--     ON media.shares(file_id, is_active) WHERE is_active = TRUE;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shares_user 
--     ON media.shares(shared_with_user_id, created_at DESC);

-- -- Stickers - Sticker usage
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_stickers_pack 
--     ON media.stickers(sticker_pack_id, is_active);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_sticker_packs 
--     ON media.user_sticker_packs(user_id, display_order);

-- -- =====================================================
-- -- NOTIFICATIONS SCHEMA INDEXES
-- -- =====================================================

-- -- Notifications - User's notifications
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_user_unread 
--     ON notifications.notifications(user_id, created_at DESC) 
--     WHERE is_read = FALSE AND deleted_at IS NULL;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_user 
--     ON notifications.notifications(user_id, created_at DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_scheduled 
--     ON notifications.notifications(scheduled_for) 
--     WHERE scheduled_for IS NOT NULL AND delivery_status = 'pending';

-- -- Push delivery
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_push_delivery 
--     ON notifications.push_delivery_log(notification_id, user_id);

-- -- Conversation channels - Muted conversations
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversation_channels 
--     ON notifications.conversation_channels(user_id, conversation_id);

-- -- Announcements
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_announcements_active 
--     ON notifications.announcements(is_active, starts_at, ends_at) 
--     WHERE is_active = TRUE;

-- -- =====================================================
-- -- ANALYTICS SCHEMA INDEXES
-- -- =====================================================

-- -- Events - User activity
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_user 
--     ON analytics.events(user_id, event_timestamp DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_name 
--     ON analytics.events(event_name, event_timestamp DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_session 
--     ON analytics.events(session_id, event_timestamp);

-- -- Daily active users
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_dau_date 
--     ON analytics.daily_active_users(date DESC, user_id);

-- -- Feature usage
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_feature_usage 
--     ON analytics.feature_usage(user_id, feature_name, date DESC);

-- -- Revenue events
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_revenue_user 
--     ON analytics.revenue_events(user_id, transaction_date DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_revenue_date 
--     ON analytics.revenue_events(transaction_date DESC) 
--     WHERE status = 'completed';

-- -- Error logs
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_error_logs 
--     ON analytics.error_logs(severity, created_at DESC) 
--     WHERE is_resolved = FALSE;

-- -- =====================================================
-- -- LOCATION SCHEMA INDEXES
-- -- =====================================================

-- -- User locations - Location history
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_locations 
--     ON location.user_locations(user_id, captured_at DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_locations_country 
--     ON location.user_locations(country, city, captured_at DESC);

-- -- IP addresses - IP lookup
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ip_addresses 
--     ON location.ip_addresses(ip_address);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ip_country 
--     ON location.ip_addresses(country_code, last_seen_at DESC);

-- -- Location shares
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_location_shares_user 
--     ON location.location_shares(user_id, is_active) 
--     WHERE is_active = TRUE;

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_location_shares_shared 
--     ON location.location_shares(shared_with_user_id, is_active);

-- -- IP blacklist
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ip_blacklist 
--     ON location.ip_blacklist(ip_address, is_active) 
--     WHERE is_active = TRUE;

-- -- =====================================================
-- -- AUDIT SCHEMA INDEXES
-- -- =====================================================

-- -- Record changes - Audit trail
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_record 
--     ON audit.record_changes(schema_name, table_name, record_id);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_user 
--     ON audit.record_changes(user_id, operation_timestamp DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_timestamp 
--     ON audit.record_changes(operation_timestamp DESC);

-- -- Auth events - Security monitoring
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_auth_user 
--     ON audit.auth_events(user_id, created_at DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_auth_type 
--     ON audit.auth_events(event_type, created_at DESC);

-- -- Admin actions - Admin activity tracking
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_admin 
--     ON audit.admin_actions(admin_user_id, created_at DESC);

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_admin_target 
--     ON audit.admin_actions(target_user_id, created_at DESC);

-- -- =====================================================
-- -- COMPOSITE INDEXES FOR COMPLEX QUERIES
-- -- =====================================================

-- -- Message search with conversation filter
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_conversation_search 
--     ON messages.messages(conversation_id, is_deleted, created_at DESC);

-- -- User contacts with status
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contacts_status_interaction 
--     ON users.contacts(user_id, status, last_interaction_at DESC);

-- -- Active sessions per user
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_user_device 
--     ON auth.sessions(user_id, device_id, last_activity_at DESC) 
--     WHERE revoked_at IS NULL;

-- -- Notification delivery tracking
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_delivery 
--     ON notifications.notifications(user_id, delivery_status, sent_at DESC);

-- -- =====================================================
-- -- MAINTENANCE QUERIES
-- -- =====================================================

-- -- Create performance schema
-- CREATE SCHEMA IF NOT EXISTS performance;

-- -- View index usage statistics
-- CREATE OR REPLACE VIEW performance.index_usage_stats AS
-- SELECT 
--     schemaname,
--     relname as tablename,
--     indexrelname as indexname,
--     idx_scan as index_scans,
--     idx_tup_read as tuples_read,
--     idx_tup_fetch as tuples_fetched,
--     pg_size_pretty(pg_relation_size(indexrelid)) as index_size
-- FROM pg_stat_user_indexes
-- ORDER BY idx_scan DESC;

-- -- View unused indexes
-- CREATE OR REPLACE VIEW performance.unused_indexes AS
-- SELECT 
--     schemaname,
--     relname as tablename,
--     indexrelname as indexname,
--     pg_size_pretty(pg_relation_size(indexrelid)) as index_size
-- FROM pg_stat_user_indexes
-- WHERE idx_scan = 0
-- AND indexrelid NOT IN (
--     SELECT indexrelid FROM pg_index WHERE indisunique OR indisprimary
-- )
-- ORDER BY pg_relation_size(indexrelid) DESC;

-- -- View missing indexes (tables with many sequential scans)
-- CREATE OR REPLACE VIEW performance.missing_indexes AS
-- SELECT 
--     schemaname,
--     relname as tablename,
--     seq_scan,
--     seq_tup_read,
--     idx_scan,
--     CASE WHEN seq_scan > 0 THEN seq_tup_read / seq_scan ELSE 0 END as avg_seq_tup_read,
--     pg_size_pretty(pg_relation_size(quote_ident(schemaname)||'.'||quote_ident(relname))) as table_size
-- FROM pg_stat_user_tables
-- WHERE seq_scan > 0
-- AND seq_tup_read / NULLIF(seq_scan, 0) > 10000
-- ORDER BY seq_tup_read DESC;

-- -- Function to reindex tables
-- CREATE OR REPLACE FUNCTION performance.reindex_table(
--     schema_name TEXT,
--     table_name TEXT
-- )
-- RETURNS void AS $$
-- BEGIN
--     EXECUTE format('REINDEX TABLE %I.%I', schema_name, table_name);
--     RAISE NOTICE 'Reindexed table %.%', schema_name, table_name;
-- END;
-- $$ LANGUAGE plpgsql;

-- -- Function to analyze table statistics
-- CREATE OR REPLACE FUNCTION performance.analyze_tables()
-- RETURNS void AS $$
-- BEGIN
--     ANALYZE auth.users;
--     ANALYZE users.profiles;
--     ANALYZE users.contacts;
--     ANALYZE messages.conversations;
--     ANALYZE messages.conversation_participants;
--     ANALYZE messages.messages;
--     ANALYZE media.files;
--     ANALYZE notifications.notifications;
--     ANALYZE analytics.events;
    
--     RAISE NOTICE 'Table statistics updated';
-- END;
-- $$ LANGUAGE plpgsql;

-- COMMENT ON FUNCTION performance.analyze_tables() IS 
-- 'Updates table statistics for the query planner. Run periodically or after bulk data changes.';

-- -- Grant permissions
-- GRANT USAGE ON SCHEMA performance TO app_user;
-- GRANT SELECT ON ALL TABLES IN SCHEMA performance TO app_user;