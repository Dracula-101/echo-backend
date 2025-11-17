# Database Schema Reference

Complete reference for Echo Backend PostgreSQL database schemas - based on ACTUAL implementation.

## Overview

Echo Backend uses PostgreSQL 15+ with a multi-schema design. Each domain has its own schema for logical separation and access control.

**Database Name**: `echo_db`

**Schemas**:
- `auth` - Authentication & session management (10 tables)
- `messages` - Conversations & messaging (20 tables)
- `users` - User profiles & relationships (13 tables)
- `media` - File storage & management
- `notifications` - Push notifications
- `location` - IP geolocation data
- `analytics` - Usage metrics

## Auth Schema

### auth.users

Core authentication table for user accounts.

```sql
CREATE TABLE auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone_number VARCHAR(20) UNIQUE,
    phone_country_code VARCHAR(5),
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    password_hash TEXT NOT NULL,
    password_salt TEXT NOT NULL,
    password_algorithm VARCHAR(50) DEFAULT 'bcrypt',
    password_last_changed_at TIMESTAMPTZ,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret TEXT,
    two_factor_backup_codes TEXT[],
    account_status VARCHAR(50) DEFAULT 'active',
    account_locked_until TIMESTAMPTZ,
    failed_login_attempts INTEGER DEFAULT 0,
    last_failed_login_at TIMESTAMPTZ,
    last_successful_login_at TIMESTAMPTZ,
    requires_password_change BOOLEAN DEFAULT FALSE,
    password_history JSONB DEFAULT '[]'::JSONB,
    security_questions JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    created_by_ip INET,
    created_by_user_agent TEXT
);
```

**Key Features**:
- Email and phone number both unique
- Password history tracking (last 5 hashes)
- Account lockout support
- Two-factor authentication (2FA) support
- Account status: active, suspended, banned, deleted, pending
- Soft delete with `deleted_at`

---

### auth.sessions

Active user sessions with comprehensive device tracking.

```sql
CREATE TABLE auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    session_token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    device_id VARCHAR(255),
    device_name VARCHAR(255),
    device_type VARCHAR(50), -- mobile, tablet, desktop, web
    device_os VARCHAR(100),
    device_os_version VARCHAR(50),
    device_model VARCHAR(100),
    device_manufacturer VARCHAR(100),
    browser_name VARCHAR(100),
    browser_version VARCHAR(50),
    user_agent TEXT,
    ip_address INET NOT NULL,
    ip_country VARCHAR(100),
    ip_region VARCHAR(100),
    ip_city VARCHAR(100),
    ip_timezone VARCHAR(100),
    ip_isp VARCHAR(255),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    is_mobile BOOLEAN DEFAULT FALSE,
    is_trusted_device BOOLEAN DEFAULT FALSE,
    fcm_token TEXT, -- Firebase Cloud Messaging
    apns_token TEXT, -- Apple Push Notification
    push_enabled BOOLEAN DEFAULT TRUE,
    session_type VARCHAR(50) DEFAULT 'user',
    expires_at TIMESTAMPTZ NOT NULL,
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    last_refresh_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,
    metadata JSONB DEFAULT '{}'::JSONB
);
```

**Key Features**:
- Full device fingerprinting
- Geolocation tracking
- Push notification token storage
- Session expiration and revocation
- Activity tracking

---

### auth.otp_verifications

One-time passwords for verification flows.

```sql
CREATE TABLE auth.otp_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    identifier VARCHAR(255) NOT NULL, -- email or phone
    identifier_type VARCHAR(20) NOT NULL, -- email, phone
    otp_code VARCHAR(10) NOT NULL,
    otp_hash TEXT NOT NULL,
    purpose VARCHAR(50) NOT NULL,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 5,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    sent_via VARCHAR(50), -- sms, email, voice
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}'::JSONB
);
```

**Purpose Values**:
- `registration` - Email/phone verification during signup
- `login` - 2FA during login
- `password_reset` - Password reset verification
- `phone_verify` - Phone number verification
- `email_verify` - Email address verification
- `2fa` - Two-factor authentication

---

### auth.oauth_providers

OAuth social login integrations.

```sql
CREATE TABLE auth.oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- google, facebook, apple, github
    provider_user_id VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    provider_username VARCHAR(255),
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    scope TEXT[],
    profile_data JSONB,
    is_primary BOOLEAN DEFAULT FALSE,
    linked_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    unlinked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);
```

**Supported Providers**:
- Google
- Facebook
- Apple
- GitHub

---

### Other Auth Tables

**auth.password_reset_tokens** - Password reset flow tracking

**auth.email_verification_tokens** - Email verification tokens

**auth.security_events** - Security audit log:
- Event types: login, logout, password_change, 2fa_enable, suspicious_activity
- Severity levels: info, warning, error, critical
- Risk scoring (0-100)

**auth.login_history** - Historical login attempts:
- Login method: password, oauth, otp, biometric, api_key
- Status: success, failed, blocked
- New device/location detection

**auth.rate_limits** - Rate limiting per user/IP:
- Action types: login, register, password_reset, api_call
- Configurable windows and thresholds

**auth.api_keys** - Service-to-service authentication:
- Scope-based permissions
- Rate limiting per key
- Expiration support

---

## Messages Schema

### messages.conversations

Conversation containers (DMs, groups, channels).

```sql
CREATE TABLE messages.conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_type VARCHAR(50) NOT NULL, -- direct, group, channel, broadcast
    title VARCHAR(255),
    description TEXT,
    avatar_url TEXT,
    creator_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE SET NULL,
    is_group BOOLEAN DEFAULT FALSE,
    is_channel BOOLEAN DEFAULT FALSE,
    is_encrypted BOOLEAN DEFAULT TRUE,
    encryption_key_id TEXT,
    max_members INTEGER,
    is_public BOOLEAN DEFAULT FALSE,
    invite_link TEXT UNIQUE,
    invite_link_expires_at TIMESTAMPTZ,
    join_approval_required BOOLEAN DEFAULT FALSE,
    who_can_send_messages VARCHAR(50) DEFAULT 'all',
    who_can_add_members VARCHAR(50) DEFAULT 'admins',
    who_can_edit_info VARCHAR(50) DEFAULT 'admins',
    who_can_pin_messages VARCHAR(50) DEFAULT 'admins',
    is_active BOOLEAN DEFAULT TRUE,
    is_archived BOOLEAN DEFAULT FALSE,
    archived_at TIMESTAMPTZ,
    member_count INTEGER DEFAULT 0,
    message_count BIGINT DEFAULT 0,
    last_message_id UUID,
    last_message_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::JSONB
);
```

**Conversation Types**:
- `direct` - One-on-one conversation
- `group` - Group chat
- `channel` - Broadcast channel (one-way)
- `broadcast` - Broadcast list

**Permission Controls**:
- Granular permissions for sending messages, adding members, editing info
- Role-based access control

---

### messages.conversation_participants

User participation in conversations.

```sql
CREATE TABLE messages.conversation_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'member', -- owner, admin, moderator, member
    nickname VARCHAR(100),
    custom_notifications BOOLEAN,
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    is_pinned BOOLEAN DEFAULT FALSE,
    pin_order INTEGER,
    is_archived BOOLEAN DEFAULT FALSE,
    last_read_message_id UUID,
    last_read_at TIMESTAMPTZ,
    unread_count INTEGER DEFAULT 0,
    mention_count INTEGER DEFAULT 0,
    can_send_messages BOOLEAN DEFAULT TRUE,
    can_send_media BOOLEAN DEFAULT TRUE,
    can_add_members BOOLEAN DEFAULT FALSE,
    can_remove_members BOOLEAN DEFAULT FALSE,
    can_edit_info BOOLEAN DEFAULT FALSE,
    can_pin_messages BOOLEAN DEFAULT FALSE,
    can_delete_messages BOOLEAN DEFAULT FALSE,
    join_method VARCHAR(50),
    invited_by_user_id UUID REFERENCES auth.users(id),
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ,
    removed_by_user_id UUID REFERENCES auth.users(id),
    removal_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(conversation_id, user_id)
);
```

**Roles**:
- `owner` - Full control
- `admin` - Manage members and settings
- `moderator` - Content moderation
- `member` - Regular participant

---

### messages.messages

Individual messages with rich features.

```sql
CREATE TABLE messages.messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    sender_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE SET NULL,
    parent_message_id UUID REFERENCES messages.messages(id) ON DELETE SET NULL,
    message_type VARCHAR(50) DEFAULT 'text',
    content TEXT,
    content_encrypted BOOLEAN DEFAULT TRUE,
    content_hash TEXT,
    format_type VARCHAR(50) DEFAULT 'plain', -- plain, markdown, html
    mentions JSONB DEFAULT '[]'::JSONB,
    hashtags TEXT[],
    links JSONB DEFAULT '[]'::JSONB,
    status VARCHAR(50) DEFAULT 'sent', -- sending, sent, delivered, read, failed
    is_edited BOOLEAN DEFAULT FALSE,
    edited_at TIMESTAMPTZ,
    edit_history JSONB DEFAULT '[]'::JSONB,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_for VARCHAR(50), -- everyone, sender
    is_pinned BOOLEAN DEFAULT FALSE,
    pinned_at TIMESTAMPTZ,
    pinned_by_user_id UUID REFERENCES auth.users(id),
    delivered_at TIMESTAMPTZ,
    delivery_count INTEGER DEFAULT 0,
    read_count INTEGER DEFAULT 0,
    is_flagged BOOLEAN DEFAULT FALSE,
    flag_reason TEXT,
    flagged_at TIMESTAMPTZ,
    flagged_by_user_id UUID REFERENCES auth.users(id),
    scheduled_at TIMESTAMPTZ,
    is_scheduled BOOLEAN DEFAULT FALSE,
    reply_count INTEGER DEFAULT 0,
    last_reply_at TIMESTAMPTZ,
    reaction_count INTEGER DEFAULT 0,
    is_forwarded BOOLEAN DEFAULT FALSE,
    forwarded_from_message_id UUID REFERENCES messages.messages(id),
    forward_count INTEGER DEFAULT 0,
    sent_from_device_id VARCHAR(255),
    sent_from_ip INET,
    expires_at TIMESTAMPTZ,
    expire_after_seconds INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
```

**Message Types**:
- `text` - Plain text
- `image` - Image with optional caption
- `video` - Video message
- `audio` - Voice message
- `document` - File attachment
- `location` - GPS coordinates
- `contact` - Shared contact
- `sticker` - Sticker/emoji
- `gif` - Animated GIF
- `poll` - Poll/survey

**Advanced Features**:
- Message threading (replies)
- Edit history tracking
- Disappearing messages
- Message scheduling
- Content moderation/flagging
- Forwarding tracking

---

### messages.reactions

Emoji reactions to messages.

```sql
CREATE TABLE messages.reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    reaction_type VARCHAR(100) NOT NULL,
    reaction_emoji VARCHAR(100),
    reaction_skin_tone VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id, reaction_type)
);
```

---

### messages.delivery_status

Per-recipient delivery tracking.

```sql
CREATE TABLE messages.delivery_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'sent', -- sent, delivered, read, failed
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    failed_reason TEXT,
    retry_count INTEGER DEFAULT 0,
    device_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id)
);
```

**Status Flow**: sent → delivered → read

---

### Additional Message Tables

- **messages.message_media** - Media attachments (references media.files)
- **messages.link_previews** - Rich link preview data
- **messages.polls** - Poll messages
- **messages.poll_options** - Poll answer choices
- **messages.poll_votes** - User poll votes
- **messages.typing_indicators** - Real-time typing status with TTL
- **messages.message_reports** - User-reported messages
- **messages.drafts** - Unsent message drafts
- **messages.bookmarks** - Saved/starred messages
- **messages.pinned_messages** - Pinned conversation messages
- **messages.conversation_invites** - Invite links
- **messages.search_index** - Full-text search optimization
- **messages.calls** - Voice/video call records
- **messages.call_participants** - Per-user call participation
- **messages.conversation_settings** - Per-conversation settings (disappearing messages, read receipts, etc.)

---

## Users Schema

### users.profiles

User profile information.

```sql
CREATE TABLE users.profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    username VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    middle_name VARCHAR(100),
    bio TEXT,
    bio_links JSONB DEFAULT '[]'::JSONB,
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
    online_status VARCHAR(20) DEFAULT 'offline',
    last_seen_at TIMESTAMPTZ,
    profile_visibility VARCHAR(20) DEFAULT 'public',
    search_visibility BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    website_url TEXT,
    social_links JSONB DEFAULT '{}'::JSONB,
    interests TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deactivated_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::JSONB
);
```

**Online Status**:
- `online` - Currently active
- `offline` - Not connected
- `away` - Idle
- `busy` - Do not disturb
- `invisible` - Appear offline

**Profile Visibility**:
- `public` - Visible to everyone
- `friends` - Visible to contacts only
- `private` - Hidden

---

### users.contacts

Friend/contact relationships.

```sql
CREATE TABLE users.contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    contact_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) DEFAULT 'contact',
    status VARCHAR(50) DEFAULT 'pending',
    nickname VARCHAR(100),
    notes TEXT,
    is_favorite BOOLEAN DEFAULT FALSE,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    custom_notifications JSONB,
    contact_source VARCHAR(50),
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
```

**Relationship Types**:
- `friend` - Mutual friendship
- `contact` - Added contact
- `blocked` - Blocked user
- `follow` - One-way follow

**Status**:
- `pending` - Friend request sent
- `accepted` - Friends
- `rejected` - Request declined
- `blocked` - User blocked

---

### users.settings

Comprehensive user preferences.

```sql
CREATE TABLE users.settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,

    -- Privacy Settings
    profile_visibility VARCHAR(20) DEFAULT 'public',
    last_seen_visibility VARCHAR(20) DEFAULT 'everyone',
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
    notification_preview VARCHAR(20) DEFAULT 'full',
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
    screen_lock_timeout INTEGER DEFAULT 0,
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
```

---

### Additional User Tables

- **users.contact_groups** - Organize contacts into groups
- **users.blocked_users** - Blocked user list (separate from contacts)
- **users.privacy_overrides** - Per-contact privacy customization
- **users.status_history** - User status/stories (temporary, WhatsApp-style)
- **users.status_views** - Who viewed user's status
- **users.activity_log** - User activity audit trail
- **users.preferences** - Key-value user preferences
- **users.devices** - Registered devices for multi-device support
- **users.achievements** - User badges/achievements (gamification)
- **users.reports** - User-reported profiles

---

## Common Patterns

### UUID Primary Keys

All tables use UUID v4:
```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid()
```

### Timestamps

Standard timestamp fields:
```sql
created_at TIMESTAMPTZ DEFAULT NOW(),
updated_at TIMESTAMPTZ DEFAULT NOW()
```

### Soft Deletes

Most tables support soft deletion:
```sql
deleted_at TIMESTAMPTZ
```

Query non-deleted records:
```sql
SELECT * FROM table WHERE deleted_at IS NULL
```

### JSONB Metadata

Flexible additional data:
```sql
metadata JSONB DEFAULT '{}'::JSONB
```

### Foreign Key Cascades

Standard cascade behavior:
- `ON DELETE CASCADE` - Delete child records
- `ON DELETE SET NULL` - Nullify reference
- `ON DELETE RESTRICT` - Prevent deletion (default)

---

**Last Updated**: January 2025
**Based on actual SQL schema files in `/database/schemas/`**
