-- =====================================================
-- USER SCHEMA - User Profiles & Relationships
-- =====================================================

-- Create Schema
CREATE SCHEMA IF NOT EXISTS users;

-- User Profiles
CREATE TABLE users.profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    username VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    middle_name VARCHAR(100),
    bio TEXT,
    bio_links JSONB DEFAULT '[]'::JSONB, -- [{url, title, icon}]
    avatar_url TEXT,
    avatar_thumbnail_url TEXT,
    cover_image_url TEXT,
    date_of_birth DATE,
    gender VARCHAR(50),
    pronouns VARCHAR(50),
    language_code VARCHAR(10) DEFAULT 'en',
    timezone VARCHAR(100),
    country_code VARCHAR(5),
    city VARCHAR(100),
    phone_visible BOOLEAN DEFAULT FALSE,
    email_visible BOOLEAN DEFAULT FALSE,
    online_status VARCHAR(20) DEFAULT 'offline', -- online, offline, away, busy, invisible
    last_seen_at TIMESTAMPTZ,
    profile_visibility VARCHAR(20) DEFAULT 'public', -- public, friends, private
    search_visibility BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    website_url TEXT,
    social_links JSONB DEFAULT '{}'::JSONB, -- {twitter, instagram, linkedin, etc}
    interests TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deactivated_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::JSONB
);

-- User Contacts/Friends
CREATE TABLE users.contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    contact_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) DEFAULT 'contact', -- friend, contact, blocked, follow
    status VARCHAR(50) DEFAULT 'pending', -- pending, accepted, rejected, blocked
    nickname VARCHAR(100),
    notes TEXT,
    is_favorite BOOLEAN DEFAULT FALSE,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    custom_notifications JSONB,
    contact_source VARCHAR(50), -- phone_contacts, search, suggestion, qr_code, link
    contact_groups TEXT[],
    last_interaction_at TIMESTAMPTZ,
    interaction_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,
    blocked_at TIMESTAMPTZ,
    block_reason TEXT,
    UNIQUE(user_id, contact_user_id),
    CHECK (user_id != contact_user_id)
);

-- User Contact Groups
CREATE TABLE users.contact_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    group_name VARCHAR(100) NOT NULL,
    group_color VARCHAR(7), -- Hex color
    group_icon VARCHAR(50),
    description TEXT,
    member_count INTEGER DEFAULT 0,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, group_name)
);

-- User Settings
CREATE TABLE users.settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Privacy Settings
    profile_visibility VARCHAR(20) DEFAULT 'public',
    last_seen_visibility VARCHAR(20) DEFAULT 'everyone', -- everyone, contacts, nobody
    online_status_visibility VARCHAR(20) DEFAULT 'everyone',
    profile_photo_visibility VARCHAR(20) DEFAULT 'everyone',
    about_visibility VARCHAR(20) DEFAULT 'everyone',
    read_receipts_enabled BOOLEAN DEFAULT TRUE,
    typing_indicators_enabled BOOLEAN DEFAULT TRUE,
    
    -- Notification Settings
    push_notifications_enabled BOOLEAN DEFAULT TRUE,
    email_notifications_enabled BOOLEAN DEFAULT TRUE,
    sms_notifications_enabled BOOLEAN DEFAULT FALSE,
    message_notifications BOOLEAN DEFAULT TRUE,
    group_message_notifications BOOLEAN DEFAULT TRUE,
    mention_notifications BOOLEAN DEFAULT TRUE,
    reaction_notifications BOOLEAN DEFAULT TRUE,
    call_notifications BOOLEAN DEFAULT TRUE,
    notification_sound VARCHAR(100) DEFAULT 'default',
    vibration_enabled BOOLEAN DEFAULT TRUE,
    notification_preview VARCHAR(20) DEFAULT 'full', -- full, name_only, none
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    
    -- Chat Settings
    enter_key_to_send BOOLEAN DEFAULT FALSE,
    auto_download_photos BOOLEAN DEFAULT TRUE,
    auto_download_videos BOOLEAN DEFAULT FALSE,
    auto_download_documents BOOLEAN DEFAULT FALSE,
    auto_download_on_wifi_only BOOLEAN DEFAULT TRUE,
    compress_images BOOLEAN DEFAULT TRUE,
    save_to_gallery BOOLEAN DEFAULT FALSE,
    chat_backup_enabled BOOLEAN DEFAULT TRUE,
    chat_backup_frequency VARCHAR(20) DEFAULT 'daily',
    
    -- Security Settings
    screen_lock_enabled BOOLEAN DEFAULT FALSE,
    screen_lock_timeout INTEGER DEFAULT 0, -- seconds
    fingerprint_unlock BOOLEAN DEFAULT FALSE,
    face_unlock BOOLEAN DEFAULT FALSE,
    show_security_notifications BOOLEAN DEFAULT TRUE,
    
    -- Display Settings
    theme VARCHAR(20) DEFAULT 'system', -- light, dark, system
    font_size VARCHAR(20) DEFAULT 'medium',
    chat_wallpaper TEXT,
    use_system_emoji BOOLEAN DEFAULT TRUE,
    
    -- Language & Region
    language_code VARCHAR(10) DEFAULT 'en',
    timezone VARCHAR(100),
    date_format VARCHAR(20) DEFAULT 'MM/DD/YYYY',
    time_format VARCHAR(20) DEFAULT '12h',
    
    -- Data Usage
    low_data_mode BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Blocked Users
CREATE TABLE users.blocked_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    blocked_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    block_reason TEXT,
    blocked_at TIMESTAMPTZ DEFAULT NOW(),
    unblocked_at TIMESTAMPTZ,
    block_type VARCHAR(50) DEFAULT 'full', -- full, messages_only, calls_only
    metadata JSONB DEFAULT '{}'::JSONB,
    UNIQUE(user_id, blocked_user_id),
    CHECK (user_id != blocked_user_id)
);

-- User Privacy Settings Override (per contact)
CREATE TABLE users.privacy_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    target_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    last_seen_visible BOOLEAN,
    online_status_visible BOOLEAN,
    profile_photo_visible BOOLEAN,
    about_visible BOOLEAN,
    read_receipts_enabled BOOLEAN,
    typing_indicators_enabled BOOLEAN,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, target_user_id)
);

-- User Status History
CREATE TABLE users.status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    status_text TEXT,
    status_emoji VARCHAR(100),
    media_url TEXT,
    media_type VARCHAR(50),
    background_color VARCHAR(7),
    views_count INTEGER DEFAULT 0,
    privacy VARCHAR(20) DEFAULT 'public', -- public, contacts, close_friends, private
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- User Status Views
CREATE TABLE users.status_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status_id UUID NOT NULL REFERENCES users.status_history(id) ON DELETE CASCADE,
    viewer_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    view_duration INTEGER, -- milliseconds
    UNIQUE(status_id, viewer_user_id)
);

-- User Activity Log
CREATE TABLE users.activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    activity_type VARCHAR(100) NOT NULL, -- profile_update, status_change, settings_change
    activity_category VARCHAR(50),
    description TEXT,
    old_value JSONB,
    new_value JSONB,
    ip_address INET,
    user_agent TEXT,
    device_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Preferences (Key-Value store for flexible settings)
CREATE TABLE users.preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    preference_key VARCHAR(255) NOT NULL,
    preference_value JSONB NOT NULL,
    category VARCHAR(100),
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, preference_key)
);

-- User Devices
CREATE TABLE users.devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    device_id VARCHAR(255) NOT NULL,
    device_name VARCHAR(255),
    device_type VARCHAR(50),
    device_model VARCHAR(100),
    device_manufacturer VARCHAR(100),
    os_name VARCHAR(100),
    os_version VARCHAR(50),
    app_version VARCHAR(50),
    is_current_device BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    last_active_at TIMESTAMPTZ DEFAULT NOW(),
    registered_at TIMESTAMPTZ DEFAULT NOW(),
    fcm_token TEXT,
    apns_token TEXT,
    push_enabled BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'::JSONB,
    UNIQUE(user_id, device_id)
);

-- User Achievements/Badges
CREATE TABLE users.achievements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    achievement_type VARCHAR(100) NOT NULL, -- early_adopter, verified, premium, milestone_messages
    achievement_name VARCHAR(255),
    achievement_description TEXT,
    achievement_icon TEXT,
    achievement_rarity VARCHAR(50), -- common, rare, epic, legendary
    progress INTEGER DEFAULT 0,
    progress_total INTEGER,
    is_unlocked BOOLEAN DEFAULT FALSE,
    unlocked_at TIMESTAMPTZ,
    display_on_profile BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Reports (when users report others)
CREATE TABLE users.reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    reported_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    report_type VARCHAR(100) NOT NULL, -- spam, harassment, inappropriate_content, impersonation
    report_category VARCHAR(100),
    description TEXT,
    evidence_urls TEXT[],
    status VARCHAR(50) DEFAULT 'pending', -- pending, reviewing, resolved, dismissed
    priority VARCHAR(20) DEFAULT 'medium',
    assigned_to UUID REFERENCES auth.users(id),
    resolution TEXT,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_profiles_user ON users.profiles(user_id);
CREATE INDEX idx_profiles_username ON users.profiles(username);
CREATE INDEX idx_contacts_user ON users.contacts(user_id);
CREATE INDEX idx_contacts_contact_user ON users.contacts(contact_user_id);
CREATE INDEX idx_contacts_status ON users.contacts(status);
CREATE INDEX idx_blocked_users_user ON users.blocked_users(user_id);
CREATE INDEX idx_blocked_users_blocked ON users.blocked_users(blocked_user_id);
CREATE INDEX idx_status_history_user ON users.status_history(user_id);
CREATE INDEX idx_status_history_expires ON users.status_history(expires_at);
CREATE INDEX idx_activity_log_user ON users.activity_log(user_id);
CREATE INDEX idx_activity_log_created ON users.activity_log(created_at);