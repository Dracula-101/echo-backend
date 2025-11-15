-- =====================================================
-- USERS SCHEMA - INDEXES (FIXED)
-- =====================================================

-- Profiles table indexes
CREATE INDEX IF NOT EXISTS idx_users_profiles_user ON users.profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_users_profiles_username ON users.profiles(username);
CREATE INDEX IF NOT EXISTS idx_users_profiles_display_name ON users.profiles(display_name);
CREATE INDEX IF NOT EXISTS idx_users_profiles_email_visible ON users.profiles(email_visible) WHERE email_visible = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_profiles_phone_visible ON users.profiles(phone_visible) WHERE phone_visible = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_profiles_online_status ON users.profiles(online_status);
CREATE INDEX IF NOT EXISTS idx_users_profiles_last_seen ON users.profiles(last_seen_at);
CREATE INDEX IF NOT EXISTS idx_users_profiles_visibility ON users.profiles(profile_visibility);
CREATE INDEX IF NOT EXISTS idx_users_profiles_search ON users.profiles(search_visibility) WHERE search_visibility = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_profiles_verified ON users.profiles(is_verified) WHERE is_verified = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_profiles_country ON users.profiles(country_code);
CREATE INDEX IF NOT EXISTS idx_users_profiles_city ON users.profiles(city);
CREATE INDEX IF NOT EXISTS idx_users_profiles_created ON users.profiles(created_at);
CREATE INDEX IF NOT EXISTS idx_users_profiles_deactivated ON users.profiles(deactivated_at) WHERE deactivated_at IS NOT NULL;

-- Full-text search index for profiles
CREATE INDEX IF NOT EXISTS idx_users_profiles_search_text ON users.profiles 
    USING GIN(to_tsvector('english', COALESCE(display_name, '') || ' ' || COALESCE(username, '') || ' ' || COALESCE(bio, '')));

-- Contacts table indexes
CREATE INDEX IF NOT EXISTS idx_users_contacts_user ON users.contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_users_contacts_contact_user ON users.contacts(contact_user_id);
CREATE INDEX IF NOT EXISTS idx_users_contacts_relationship ON users.contacts(relationship_type);
CREATE INDEX IF NOT EXISTS idx_users_contacts_status ON users.contacts(status);
CREATE INDEX IF NOT EXISTS idx_users_contacts_favorite ON users.contacts(user_id, is_favorite) WHERE is_favorite = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_contacts_pinned ON users.contacts(user_id, is_pinned) WHERE is_pinned = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_contacts_archived ON users.contacts(is_archived);
CREATE INDEX IF NOT EXISTS idx_users_contacts_muted ON users.contacts(is_muted) WHERE is_muted = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_contacts_source ON users.contacts(contact_source);
CREATE INDEX IF NOT EXISTS idx_users_contacts_last_interaction ON users.contacts(last_interaction_at);
CREATE INDEX IF NOT EXISTS idx_users_contacts_created ON users.contacts(created_at);

-- Contact groups table indexes
CREATE INDEX IF NOT EXISTS idx_users_contact_groups_user ON users.contact_groups(user_id);
CREATE INDEX IF NOT EXISTS idx_users_contact_groups_name ON users.contact_groups(user_id, group_name);
CREATE INDEX IF NOT EXISTS idx_users_contact_groups_default ON users.contact_groups(user_id, is_default) WHERE is_default = TRUE;

-- Settings table indexes
CREATE INDEX IF NOT EXISTS idx_users_settings_user ON users.settings(user_id);

-- Blocked users table indexes
CREATE INDEX IF NOT EXISTS idx_users_blocked_user ON users.blocked_users(user_id);
CREATE INDEX IF NOT EXISTS idx_users_blocked_blocked_user ON users.blocked_users(blocked_user_id);
CREATE INDEX IF NOT EXISTS idx_users_blocked_at ON users.blocked_users(blocked_at);
CREATE INDEX IF NOT EXISTS idx_users_blocked_unblocked ON users.blocked_users(unblocked_at) WHERE unblocked_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_blocked_type ON users.blocked_users(block_type);

-- Privacy overrides table indexes
CREATE INDEX IF NOT EXISTS idx_users_privacy_overrides_user ON users.privacy_overrides(user_id);
CREATE INDEX IF NOT EXISTS idx_users_privacy_overrides_target ON users.privacy_overrides(target_user_id);

-- Status history table indexes
CREATE INDEX IF NOT EXISTS idx_users_status_history_user ON users.status_history(user_id);
CREATE INDEX IF NOT EXISTS idx_users_status_history_created ON users.status_history(created_at);
CREATE INDEX IF NOT EXISTS idx_users_status_history_expires ON users.status_history(expires_at);
CREATE INDEX IF NOT EXISTS idx_users_status_history_deleted ON users.status_history(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status_history_privacy ON users.status_history(privacy);
-- FIXED: Removed NOW() from index predicate
CREATE INDEX IF NOT EXISTS idx_users_status_history_active ON users.status_history(user_id, expires_at) 
    WHERE deleted_at IS NULL;

-- Status views table indexes
CREATE INDEX IF NOT EXISTS idx_users_status_views_status ON users.status_views(status_id);
CREATE INDEX IF NOT EXISTS idx_users_status_views_viewer ON users.status_views(viewer_user_id);
CREATE INDEX IF NOT EXISTS idx_users_status_views_viewed_at ON users.status_views(viewed_at);

-- Activity log table indexes
CREATE INDEX IF NOT EXISTS idx_users_activity_log_user ON users.activity_log(user_id);
CREATE INDEX IF NOT EXISTS idx_users_activity_log_type ON users.activity_log(activity_type);
CREATE INDEX IF NOT EXISTS idx_users_activity_log_category ON users.activity_log(activity_category);
CREATE INDEX IF NOT EXISTS idx_users_activity_log_created ON users.activity_log(created_at);
CREATE INDEX IF NOT EXISTS idx_users_activity_log_ip ON users.activity_log(ip_address);

-- Preferences table indexes
CREATE INDEX IF NOT EXISTS idx_users_preferences_user ON users.preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_users_preferences_key ON users.preferences(preference_key);
CREATE INDEX IF NOT EXISTS idx_users_preferences_category ON users.preferences(category);
CREATE INDEX IF NOT EXISTS idx_users_preferences_system ON users.preferences(is_system) WHERE is_system = TRUE;

-- Devices table indexes
CREATE INDEX IF NOT EXISTS idx_users_devices_user ON users.devices(user_id);
CREATE INDEX IF NOT EXISTS idx_users_devices_device_id ON users.devices(device_id);
CREATE INDEX IF NOT EXISTS idx_users_devices_current ON users.devices(user_id, is_current_device) WHERE is_current_device = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_devices_active ON users.devices(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_devices_last_active ON users.devices(last_active_at);
CREATE INDEX IF NOT EXISTS idx_users_devices_push_enabled ON users.devices(push_enabled) WHERE push_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_devices_fcm_token ON users.devices(fcm_token) WHERE fcm_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_devices_apns_token ON users.devices(apns_token) WHERE apns_token IS NOT NULL;

-- Achievements table indexes
CREATE INDEX IF NOT EXISTS idx_users_achievements_user ON users.achievements(user_id);
CREATE INDEX IF NOT EXISTS idx_users_achievements_type ON users.achievements(achievement_type);
CREATE INDEX IF NOT EXISTS idx_users_achievements_unlocked ON users.achievements(is_unlocked) WHERE is_unlocked = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_achievements_display ON users.achievements(user_id, display_on_profile) WHERE display_on_profile = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_achievements_rarity ON users.achievements(achievement_rarity);

-- Reports table indexes
CREATE INDEX IF NOT EXISTS idx_users_reports_reporter ON users.reports(reporter_user_id);
CREATE INDEX IF NOT EXISTS idx_users_reports_reported ON users.reports(reported_user_id);
CREATE INDEX IF NOT EXISTS idx_users_reports_type ON users.reports(report_type);
CREATE INDEX IF NOT EXISTS idx_users_reports_status ON users.reports(status);
CREATE INDEX IF NOT EXISTS idx_users_reports_priority ON users.reports(priority);
CREATE INDEX IF NOT EXISTS idx_users_reports_assigned ON users.reports(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_reports_created ON users.reports(created_at);
CREATE INDEX IF NOT EXISTS idx_users_reports_resolved ON users.reports(resolved_at) WHERE resolved_at IS NOT NULL;