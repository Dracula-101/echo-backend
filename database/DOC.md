# Echo Backend - Database Documentation

> **Version:** 1.0  
> **Last Updated:** November 3, 2025  
> **Database System:** PostgreSQL 14+

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Schemas](#schemas)
4. [Database Objects](#database-objects)
5. [Security & Access Control](#security--access-control)
6. [Performance Optimization](#performance-optimization)
7. [Data Integrity & Triggers](#data-integrity--triggers)
8. [Analytics & Reporting](#analytics--reporting)
9. [Maintenance & Operations](#maintenance--operations)

---

## Overview

The Echo Backend database is a comprehensive, enterprise-grade PostgreSQL database designed to support a real-time messaging and communication platform. It implements a microservices-oriented schema design with strong data integrity, security, and performance optimization.

### Key Features

- **Multi-schema architecture** for logical separation
- **Row-Level Security (RLS)** for data protection
- **Comprehensive audit trails** for compliance
- **Real-time capabilities** for messaging and presence
- **Analytics and reporting** views for business intelligence
- **Geolocation tracking** for user locations and IP addresses
- **Media management** with cloud storage integration (R2/S3)
- **End-to-end encryption support** for secure communications

### Technology Stack

- **Database:** PostgreSQL 14+
- **Extensions:** pgcrypto, uuid-ossp, pg_trgm (text search)
- **Storage:** Cloudflare R2 / AWS S3 (referenced)
- **Caching:** Redis (application layer)

---

## Architecture

### Schema Organization

The database is organized into 7 primary schemas, each serving a specific domain:

```
echo-backend (database)
├── auth          (Authentication & Sessions)
├── users         (User Profiles & Relationships)
├── messages      (Conversations & Messages)
├── media         (File Storage & Processing)
├── notifications (Push & In-App Notifications)
├── analytics     (User Behavior & Metrics)
├── location      (IP Tracking & Geolocation)
└── audit         (Change Tracking & Compliance)
```

### Design Principles

1. **Schema Isolation:** Each business domain has its own schema for logical separation
2. **UUID Primary Keys:** All tables use UUID for distributed system compatibility
3. **Soft Deletes:** Important records use `deleted_at` instead of hard deletes
4. **Timestamp Tracking:** All tables include `created_at` and `updated_at`
5. **JSONB Flexibility:** Metadata fields for extensibility without schema changes
6. **Referential Integrity:** Foreign keys with appropriate ON DELETE actions

---

## Schemas

### 1. Auth Schema

**Purpose:** Manages user authentication, sessions, and security.

#### Core Tables

##### `auth.users`
Primary authentication table storing user credentials and security settings.

**Key Columns:**
- `id` (UUID, PK) - Unique user identifier
- `email` (VARCHAR, UNIQUE) - User email address
- `phone_number` (VARCHAR, UNIQUE) - User phone number
- `password_hash` (TEXT) - Bcrypt hashed password
- `two_factor_enabled` (BOOLEAN) - 2FA status
- `account_status` (VARCHAR) - active, suspended, banned, deleted, pending
- `failed_login_attempts` (INTEGER) - Failed login counter for security

**Relationships:**
- Referenced by: `users.profiles`, `auth.sessions`, `messages.messages`, etc.

##### `auth.sessions`
Tracks active user sessions with device and location information.

**Key Columns:**
- `session_token` (TEXT, UNIQUE) - JWT or session identifier
- `refresh_token` (TEXT, UNIQUE) - Token refresh capability
- `device_type` (VARCHAR) - mobile, tablet, desktop, web
- `ip_address` (INET) - Session IP address
- `fcm_token` (TEXT) - Firebase Cloud Messaging token
- `expires_at` (TIMESTAMPTZ) - Session expiration
- `revoked_at` (TIMESTAMPTZ) - Manual session termination

**Features:**
- Device fingerprinting for security
- Geographic tracking via IP
- Push notification token storage
- Automatic expiration handling

##### `auth.otp_verifications`
One-Time Password management for multi-factor authentication.

**Key Columns:**
- `identifier` (VARCHAR) - Email or phone
- `otp_code` (VARCHAR) - Plain OTP (temporary)
- `otp_hash` (TEXT) - Hashed OTP for security
- `purpose` (VARCHAR) - registration, login, password_reset, 2fa
- `attempts` (INTEGER) - Failed verification attempts
- `expires_at` (TIMESTAMPTZ) - OTP expiration time

##### `auth.oauth_providers`
OAuth integration for social login (Google, Facebook, Apple, etc.).

##### `auth.security_events`
Audit log for security-related activities.

##### `auth.login_history`
Complete login attempt history with device and location.

##### `auth.api_keys`
API key management for programmatic access.

---

### 2. Users Schema

**Purpose:** User profiles, contacts/friends, and social relationships.

#### Core Tables

##### `users.profiles`
Extended user profile information.

**Key Columns:**
- `user_id` (UUID, FK → auth.users) - Links to auth user
- `username` (VARCHAR, UNIQUE) - Unique username
- `display_name` (VARCHAR) - Public display name
- `bio` (TEXT) - User biography
- `avatar_url` (TEXT) - Profile picture URL
- `online_status` (VARCHAR) - online, offline, away, busy, invisible
- `profile_visibility` (VARCHAR) - public, friends, private
- `is_verified` (BOOLEAN) - Verified account badge

**Features:**
- Multi-language support (`language_code`)
- Timezone tracking
- Social media links (JSONB)
- Interests and bio links
- Deactivation support

##### `users.contacts`
Friend/contact relationships between users.

**Key Columns:**
- `user_id` (UUID, FK → auth.users) - User requesting/holding contact
- `contact_user_id` (UUID, FK → auth.users) - Target contact
- `relationship_type` (VARCHAR) - friend, contact, blocked, follow
- `status` (VARCHAR) - pending, accepted, rejected, blocked
- `is_favorite` (BOOLEAN) - Star/favorite contact
- `is_muted` (BOOLEAN) - Mute notifications

**Features:**
- Mutual contact detection
- Contact grouping
- Interaction tracking
- Block functionality
- Custom nicknames

##### `users.contact_groups`
User-defined contact organization (Family, Work, etc.).

##### `users.settings`
User preferences and privacy settings.

**Key Columns:**
- Privacy controls (profile, last seen, online status visibility)
- `read_receipts_enabled` (BOOLEAN)
- `typing_indicators_enabled` (BOOLEAN)
- Notification preferences
- Theme and appearance settings

##### `users.blocked_users`
Dedicated table for blocking relationships.

##### `users.status_history`
WhatsApp-style status/stories feature.

##### `users.activity_log`
User activity tracking for profile changes and actions.

##### `users.devices`
User device registry for multi-device support.

---

### 3. Messages Schema

**Purpose:** Real-time messaging, conversations, calls, and reactions.

#### Core Tables

##### `messages.conversations`
Chat rooms/conversations (direct, group, channel).

**Key Columns:**
- `conversation_type` (VARCHAR) - direct, group, channel, broadcast
- `is_encrypted` (BOOLEAN) - E2E encryption flag
- `is_public` (BOOLEAN) - Public accessibility
- `member_count` (INTEGER) - Participant count
- `message_count` (BIGINT) - Total messages
- `last_message_at` (TIMESTAMPTZ) - Last activity time

**Features:**
- Group/channel management
- Invite links with expiration
- Permission controls (who can send, add members, etc.)
- Archive support

##### `messages.conversation_participants`
Many-to-many relationship between users and conversations.

**Key Columns:**
- `role` (VARCHAR) - owner, admin, moderator, member
- `is_muted` (BOOLEAN) - Mute notifications
- `is_pinned` (BOOLEAN) - Pin conversation
- `unread_count` (INTEGER) - Unread message counter
- `last_read_message_id` (UUID) - Read receipt tracking
- `can_send_messages` (BOOLEAN) - Granular permissions

**Features:**
- Custom nicknames per conversation
- Individual notification settings
- Read receipt tracking
- Join/leave history

##### `messages.messages`
Core message content table.

**Key Columns:**
- `sender_user_id` (UUID, FK → auth.users)
- `conversation_id` (UUID, FK → messages.conversations)
- `message_type` (VARCHAR) - text, image, video, audio, document, location, etc.
- `content` (TEXT) - Message content (encrypted if E2E)
- `content_encrypted` (BOOLEAN)
- `reply_to_message_id` (UUID) - Threading support
- `forward_from_message_id` (UUID) - Message forwarding
- `is_edited` (BOOLEAN) - Edit tracking
- `is_deleted` (BOOLEAN) - Soft delete
- `mentions` (JSONB) - User mentions

**Features:**
- Rich media support
- Message reactions
- Read receipts
- Edit history
- Forward tracking
- Link preview metadata

##### `messages.message_reactions`
Emoji reactions on messages (like Discord/Slack).

##### `messages.message_edits`
Complete edit history for messages.

##### `messages.message_receipts`
Detailed read/delivery status per participant.

##### `messages.calls`
Voice/video call metadata.

**Key Columns:**
- `call_type` (VARCHAR) - audio, video, screen_share
- `call_status` (VARCHAR) - ringing, ongoing, ended, missed, declined
- `duration_seconds` (INTEGER)
- `quality_rating` (INTEGER) - User feedback

##### `messages.call_participants`
Participants in group calls.

##### `messages.pinned_messages`
Pinned messages in conversations.

##### `messages.drafts`
Auto-saved message drafts.

---

### 4. Media Schema

**Purpose:** File storage, processing, and media management.

#### Core Tables

##### `media.files`
Central media file registry with cloud storage references.

**Key Columns:**
- `uploader_user_id` (UUID, FK → auth.users)
- `file_name` (VARCHAR) - Stored filename
- `file_type` (VARCHAR) - MIME type (image/jpeg, video/mp4, etc.)
- `file_size_bytes` (BIGINT)
- `storage_provider` (VARCHAR) - r2, s3, local
- `storage_key` (TEXT) - Cloud storage object key
- `storage_url` (TEXT) - Full access URL
- `cdn_url` (TEXT) - CDN-accelerated URL

**Media Metadata:**
- Image: `width`, `height`, `dominant_colors`, `has_alpha_channel`
- Video: `duration_seconds`, `codec`, `resolution`, `frame_rate`
- Audio: `audio_channels`, `sample_rate`, `bitrate`
- Document: `page_count`, `word_count`

**Features:**
- Automatic thumbnail generation (small, medium, large)
- Content hash for deduplication
- Virus scanning integration
- AI content moderation (NSFW detection)
- Encryption support
- Access control (visibility, expiration)
- Download/view tracking

##### `media.thumbnails`
Dedicated thumbnail tracking.

##### `media.processing_queue`
Asynchronous media processing jobs.

##### `media.albums`
User photo albums/collections.

##### `media.album_items`
Photos/videos in albums.

##### `media.shared_files`
File sharing with expiration and permissions.

##### `media.file_versions`
Version history for files.

##### `media.file_comments`
Commenting on media files.

---

### 5. Notifications Schema

**Purpose:** Push notifications, in-app alerts, and notification preferences.

#### Core Tables

##### `notifications.notifications`
Central notification registry.

**Key Columns:**
- `notification_type` (VARCHAR) - message, mention, reaction, call, friend_request
- `notification_category` (VARCHAR) - social, system, marketing, security
- `title` (VARCHAR) - Notification title
- `body` (TEXT) - Notification content
- `is_read` (BOOLEAN)
- `priority` (VARCHAR) - low, normal, high, urgent
- `action_url` (TEXT) - Deep link URL
- `group_key` (VARCHAR) - Notification bundling

**Features:**
- Rich media (icons, images)
- Related entity references
- Scheduled notifications
- Delivery status tracking
- Cross-platform support (iOS, Android, Web)

##### `notifications.push_delivery_log`
Push notification delivery tracking and metrics.

**Key Columns:**
- `push_provider` (VARCHAR) - fcm, apns
- `status` (VARCHAR) - pending, sent, delivered, failed
- `time_to_deliver_ms` (INTEGER) - Performance metrics
- `opened_at` (TIMESTAMPTZ) - User engagement tracking

##### `notifications.preferences`
User notification preferences per type/category.

##### `notifications.notification_queue`
Pending notifications queue for batch processing.

##### `notifications.email_notifications`
Email notification tracking.

##### `notifications.sms_notifications`
SMS notification tracking.

##### `notifications.templates`
Notification templates for consistency.

---

### 6. Analytics Schema

**Purpose:** Business intelligence, user behavior tracking, and metrics.

#### Core Tables

##### `analytics.events`
Detailed event tracking (app_open, message_sent, profile_view, etc.).

**Key Columns:**
- `event_name` (VARCHAR) - Action identifier
- `event_category` (VARCHAR) - engagement, conversion, retention
- `user_id` (UUID) - Actor
- `session_id` (UUID) - Session context
- `properties` (JSONB) - Custom event properties
- Platform and device information
- Geographic data
- Performance metrics (duration, load time)

**Features:**
- UTM tracking for attribution
- Custom properties (JSONB)
- Network quality tracking
- Performance monitoring

##### `analytics.user_sessions`
Aggregated session analytics.

##### `analytics.daily_metrics`
Pre-aggregated daily statistics for reporting.

##### `analytics.user_retention`
Cohort analysis and retention metrics.

##### `analytics.funnel_events`
Conversion funnel tracking.

##### `analytics.ab_tests`
A/B test experiment tracking.

##### `analytics.feature_usage`
Feature adoption and usage tracking.

##### `analytics.crash_reports`
Application crash and error tracking.

---

### 7. Location Schema

**Purpose:** IP geolocation, user location history, and geographic analytics.

#### Core Tables

##### `location.ip_addresses`
IP address geolocation database.

**Key Columns:**
- `ip_address` (INET, UNIQUE)
- Geographic data (country, region, city, coordinates)
- Network info (ISP, ASN, organization)
- Security flags (`is_proxy`, `is_vpn`, `is_tor`, `threat_level`)
- `risk_score` (INTEGER, 0-100)
- Usage tracking

**Features:**
- IP version support (IPv4/IPv6)
- Connection type detection
- Anonymous proxy detection
- Threat scoring

##### `location.user_locations`
User location history and tracking.

**Key Columns:**
- `location_type` (VARCHAR) - ip_based, gps, wifi, cell_tower
- Coordinates with accuracy
- Movement detection flags
- Distance calculation from previous location

##### `location.suspicious_locations`
Flagged locations for security analysis.

##### `location.geofences`
Geographic boundaries for location-based features.

##### `location.location_events`
Location-triggered events.

---

### 8. Audit Schema

**Purpose:** Compliance, security auditing, and change tracking.

#### Core Tables

##### `audit.record_changes`
Generic audit trail for all table changes.

**Key Columns:**
- `schema_name`, `table_name` - Target table
- `operation` (VARCHAR) - INSERT, UPDATE, DELETE
- `old_data`, `new_data` (JSONB) - Before/after snapshots
- `changed_fields` (TEXT[]) - Modified columns
- User and session context
- Transaction ID for atomicity

##### `audit.sensitive_data_access`
Tracks access to sensitive data (PII, credentials).

##### `audit.auth_events`
Authentication event audit trail.

##### `audit.admin_actions`
Administrative action logging.

##### `audit.data_exports`
Data export requests and GDPR compliance.

---

## Database Objects

### Functions

#### Authentication & Security Functions

**`auth.current_user_id()` → UUID**
- Returns the currently authenticated user ID from session context
- Used extensively in RLS policies

**`auth.is_admin()` → BOOLEAN**
- Checks if current user has admin privileges

**`auth.hash_password(password TEXT)` → TEXT**
- Hashes password using bcrypt with salt

**`auth.verify_password(password TEXT, hash TEXT)` → BOOLEAN**
- Verifies password against stored hash

**`auth.generate_otp(length INTEGER)` → VARCHAR**
- Generates random numeric OTP code

**`auth.generate_token(length INTEGER)` → TEXT**
- Generates secure random token (hex encoded)

#### Message & Conversation Functions

**`messages.is_conversation_participant(conv_id UUID)` → BOOLEAN**
- Checks if current user is a participant in conversation

**`messages.is_conversation_admin(conv_id UUID)` → BOOLEAN**
- Checks if current user is admin/owner of conversation

**`messages.can_send_messages(conv_id UUID)` → BOOLEAN**
- Checks if current user has permission to send messages

**`messages.can_delete_message(msg_id UUID)` → BOOLEAN**
- Checks if current user can delete a specific message

#### User & Profile Functions

**`users.are_contacts(user_id_a UUID, user_id_b UUID)` → BOOLEAN**
- Checks if two users are connected as contacts

**`users.is_blocked_by(target_user_id UUID)` → BOOLEAN**
- Checks if current user is blocked by target user

**`users.generate_unique_username(base_name VARCHAR)` → VARCHAR**
- Generates unique username from base name

#### Location Functions

**`location.calculate_distance(lat1, lon1, lat2, lon2)` → DECIMAL**
- Calculates distance in kilometers using Haversine formula

#### Media Functions

**`media.format_file_size(bytes BIGINT)` → VARCHAR**
- Formats byte size to human-readable format (KB, MB, GB)

---

### Triggers

#### Update Timestamp Triggers
Applied to all tables to automatically update `updated_at` column.

```sql
CREATE TRIGGER trigger_<table>_updated_at
    BEFORE UPDATE ON <schema>.<table>
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

#### Security Event Logging
**`auth.log_security_event()`**
- Triggered on: `auth.users` UPDATE
- Actions: Logs password changes, 2FA toggles, account status changes

#### Session Management
**`auth.revoke_expired_session()`**
- Triggered on: `auth.sessions` UPDATE
- Actions: Auto-revokes expired sessions

#### Profile Activity Tracking
**`users.log_profile_changes()`**
- Triggered on: `users.profiles` UPDATE
- Actions: Logs username, display name, avatar changes

#### Mutual Contact Auto-Accept
**`users.handle_mutual_contact_request()`**
- Triggered on: `users.contacts` INSERT
- Actions: Auto-accepts when both users request each other

#### Conversation Stats Updates
**`messages.update_conversation_stats()`**
- Triggered on: `messages.messages` INSERT
- Actions: Increments message count, updates last message info

#### Unread Count Management
**`messages.update_unread_counts()`**
- Triggered on: `messages.messages` INSERT
- Actions: Increments unread count for all participants except sender

#### Media Processing Queue
**`media.queue_for_processing()`**
- Triggered on: `media.files` INSERT
- Actions: Adds new files to processing queue

#### Audit Trail Recording
**`audit.record_change()`**
- Triggered on: ALL monitored tables (INSERT/UPDATE/DELETE)
- Actions: Records complete change history with before/after data

---

### Indexes

Performance indexes are strategically placed for common query patterns.

#### Auth Schema Indexes

```sql
-- Fast login lookups
idx_users_email_active ON auth.users(email) WHERE deleted_at IS NULL
idx_users_phone ON auth.users(phone_number) WHERE phone_verified = TRUE

-- Active session queries
idx_sessions_user_active ON auth.sessions(user_id, last_activity_at DESC)
idx_sessions_token ON auth.sessions(session_token) WHERE revoked_at IS NULL

-- OTP verification
idx_otp_identifier ON auth.otp_verifications(identifier, identifier_type, expires_at DESC)
```

#### Users Schema Indexes

```sql
-- Profile search (full-text search)
idx_profiles_search ON users.profiles USING gin(to_tsvector('english', ...))
idx_profiles_username ON users.profiles(LOWER(username))

-- Contact relationships
idx_contacts_user_status ON users.contacts(user_id, status)
idx_contacts_mutual ON users.contacts(contact_user_id, user_id, status)

-- Block checking
idx_blocked_users_check ON users.blocked_users(user_id, blocked_user_id)
```

#### Messages Schema Indexes

```sql
-- Message queries
idx_messages_conversation ON messages.messages(conversation_id, created_at DESC)
idx_messages_sender ON messages.messages(sender_user_id, created_at DESC)

-- Unread messages
idx_messages_unread ON messages.messages(conversation_id, id) WHERE is_deleted = FALSE

-- Participant lookups
idx_participants_user ON messages.conversation_participants(user_id)
idx_participants_conversation ON messages.conversation_participants(conversation_id)
```

#### Media Schema Indexes

```sql
-- File queries
idx_files_uploader ON media.files(uploader_user_id, created_at DESC)
idx_files_hash ON media.files(content_hash) -- Deduplication

-- Processing queue
idx_files_processing ON media.files(processing_status) WHERE processing_status != 'completed'
```

#### Analytics Schema Indexes

```sql
-- Event tracking
idx_events_user ON analytics.events(user_id, created_at DESC)
idx_events_session ON analytics.events(session_id, event_timestamp)
idx_events_name ON analytics.events(event_name, created_at DESC)
```

#### Location Schema Indexes

```sql
-- IP lookups
idx_ip_address ON location.ip_addresses(ip_address)

-- User location history
idx_user_locations_user ON location.user_locations(user_id, captured_at DESC)

-- Geographic queries
idx_user_locations_country ON location.user_locations(country, captured_at DESC)
```

---

### Views

#### User Analytics Views

**`views.user_growth`**
- Daily user registration trends with cumulative totals

**`views.active_users_summary`**
- DAU, WAU, MAU metrics with engagement ratios

**`views.user_engagement`**
- User engagement levels (power_user, active_user, casual_user, inactive_user)

#### Message Analytics Views

**`views.messaging_stats`**
- Daily messaging activity, media sharing, conversation creation

**`views.conversation_analytics`**
- Per-conversation metrics (message count, participants, activity)

#### Media Analytics Views

**`views.media_storage_summary`**
- Storage usage by user, file type, and time period

#### Performance Views

**`views.slow_queries`**
- Database performance monitoring

---

## Security & Access Control

### Row-Level Security (RLS)

RLS policies enforce data access at the database level, complementing application-level security.

#### Implementation Pattern

```sql
-- Enable RLS on table
ALTER TABLE <schema>.<table> ENABLE ROW LEVEL SECURITY;

-- Create policy
CREATE POLICY <policy_name> ON <schema>.<table>
    FOR <operation>  -- SELECT, INSERT, UPDATE, DELETE, ALL
    USING (<condition>)           -- Read policy
    WITH CHECK (<condition>);     -- Write policy
```

#### Common Policies

**User owns data:**
```sql
CREATE POLICY users_own_profile ON users.profiles
    FOR ALL USING (user_id = auth.current_user_id());
```

**Conversation participants:**
```sql
CREATE POLICY participants_read_messages ON messages.messages
    FOR SELECT USING (messages.is_conversation_participant(conversation_id));
```

**Contact visibility:**
```sql
CREATE POLICY contacts_visibility ON users.profiles
    FOR SELECT USING (
        profile_visibility = 'public'
        OR user_id = auth.current_user_id()
        OR (profile_visibility = 'friends' AND users.is_contact_with(user_id))
    );
```

**Admin override:**
```sql
CREATE POLICY admin_full_access ON <table>
    FOR ALL USING (auth.is_admin());
```

### Authentication Context

The database receives user context via PostgreSQL session variables:

```sql
-- Set in application before queries
SET app.current_user_id = '<user_uuid>';
SET app.session_id = '<session_uuid>';
```

Functions like `auth.current_user_id()` retrieve this context for RLS policies.

### Data Encryption

- **At Rest:** PostgreSQL transparent data encryption (TDE) or disk-level encryption
- **In Transit:** SSL/TLS connections enforced
- **Application Layer:** E2E encryption for messages (AES-256)
  - `messages.messages.content_encrypted` flag indicates encrypted content
  - Encryption keys managed by application, not stored in database

---

## Performance Optimization

### Query Optimization Strategies

1. **Composite Indexes:** Multi-column indexes for common query patterns
2. **Partial Indexes:** Filter conditions in index for smaller, faster indexes
3. **Covering Indexes:** Include frequently queried columns in index
4. **GIN Indexes:** Full-text search on JSONB and text columns
5. **BRIN Indexes:** Range queries on timestamp columns for large tables

### Partitioning Strategy

Large tables should be partitioned for better performance:

#### Time-Based Partitioning Example

```sql
-- Analytics events partitioned by month
CREATE TABLE analytics.events (
    -- columns
) PARTITION BY RANGE (created_at);

CREATE TABLE analytics.events_2025_11 PARTITION OF analytics.events
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
```

**Recommended partitioning:**
- `analytics.events` - Monthly partitions
- `audit.record_changes` - Monthly partitions
- `location.user_locations` - Quarterly partitions
- `messages.messages` - Yearly partitions (after 10M+ messages)

### Connection Pooling

Use PgBouncer or similar for connection pooling:
- **Pool mode:** Transaction
- **Max connections:** 100-200 per service
- **Connection timeout:** 30 seconds

### Query Caching

Application-level caching with Redis:
- User profiles (TTL: 5 minutes)
- Conversation metadata (TTL: 1 minute)
- Active sessions (TTL: session duration)
- Media URLs (TTL: 1 hour)

---

## Data Integrity & Triggers

### Referential Integrity

Foreign key constraints ensure data consistency:

```sql
-- Cascade delete: Remove dependent records
user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE

-- Set null: Preserve record, nullify reference
uploader_user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL

-- Restrict: Prevent deletion if references exist
creator_user_id UUID REFERENCES auth.users(id) ON DELETE RESTRICT
```

### Constraints

#### Check Constraints

```sql
-- Prevent self-referencing
CHECK (user_id != contact_user_id)

-- Value range validation
CHECK (rating >= 1 AND rating <= 5)

-- Enum-like constraints
CHECK (status IN ('pending', 'accepted', 'rejected'))
```

#### Unique Constraints

```sql
-- Single column
UNIQUE(username)

-- Composite uniqueness
UNIQUE(user_id, contact_user_id)

-- Conditional uniqueness (partial unique index)
CREATE UNIQUE INDEX idx_active_sessions 
ON auth.sessions(user_id, device_id) 
WHERE revoked_at IS NULL;
```

---

## Analytics & Reporting

### Key Metrics

#### User Metrics
- **DAU/WAU/MAU:** Active user counts
- **Stickiness:** DAU/MAU ratio (target: >20%)
- **Retention:** Cohort retention rates
- **Churn:** User deactivation rates

#### Engagement Metrics
- **Messages per DAU:** Average daily messages
- **Session duration:** Average time in app
- **Feature adoption:** Usage of key features
- **Call metrics:** Call duration, quality ratings

#### Growth Metrics
- **New registrations:** Daily/weekly/monthly
- **Viral coefficient:** Invites per user
- **Conversion rate:** Registration to active user
- **Time to value:** Registration to first message

### Reporting Queries

#### Active Users Report
```sql
SELECT * FROM views.active_users_summary;
```

#### Top Conversations
```sql
SELECT 
    c.id,
    c.title,
    c.member_count,
    c.message_count,
    c.last_message_at
FROM messages.conversations c
WHERE c.last_activity_at >= NOW() - INTERVAL '7 days'
ORDER BY c.message_count DESC
LIMIT 100;
```

#### User Engagement Distribution
```sql
SELECT 
    engagement_level,
    COUNT(*) as user_count,
    ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER (), 2) as percentage
FROM views.user_engagement
GROUP BY engagement_level;
```

---

## Maintenance & Operations

### Backup Strategy

**Frequency:**
- Full backup: Daily at 2 AM UTC
- Incremental backup: Every 6 hours
- WAL archiving: Continuous

**Retention:**
- Daily backups: 30 days
- Weekly backups: 3 months
- Monthly backups: 1 year

**Tools:**
- `pg_dump` for logical backups
- `pg_basebackup` for physical backups
- WAL-E or pgBackRest for continuous archiving

### Routine Maintenance

#### Vacuum & Analyze

```sql
-- Regular vacuum (automated by autovacuum)
VACUUM ANALYZE auth.users;

-- Full vacuum (requires downtime, rarely needed)
VACUUM FULL auth.sessions;

-- Analyze statistics for query planner
ANALYZE messages.messages;
```

#### Index Maintenance

```sql
-- Rebuild fragmented indexes
REINDEX INDEX idx_messages_conversation;

-- Rebuild all table indexes
REINDEX TABLE messages.messages;
```

#### Statistics Update

```sql
-- Update column statistics for better query plans
ALTER TABLE messages.messages 
ALTER COLUMN conversation_id SET STATISTICS 1000;

ANALYZE messages.messages;
```

### Monitoring

#### Key Metrics to Monitor

1. **Connection count:** Should be < max_connections
2. **Cache hit ratio:** Target >99%
3. **Transaction rate:** Queries per second
4. **Replication lag:** Should be <1 second
5. **Table bloat:** Regular vacuum monitoring
6. **Slow queries:** Log queries >1 second

#### Monitoring Queries

**Active connections:**
```sql
SELECT count(*) FROM pg_stat_activity WHERE state = 'active';
```

**Cache hit ratio:**
```sql
SELECT 
    sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) as cache_hit_ratio
FROM pg_statio_user_tables;
```

**Blocking queries:**
```sql
SELECT * FROM pg_stat_activity WHERE wait_event IS NOT NULL;
```

**Table sizes:**
```sql
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 20;
```

### Data Retention Policies

#### Automated Cleanup

**Expired sessions:**
```sql
DELETE FROM auth.sessions 
WHERE expires_at < NOW() - INTERVAL '30 days';
```

**Old OTPs:**
```sql
DELETE FROM auth.otp_verifications 
WHERE created_at < NOW() - INTERVAL '1 day';
```

**Processed media queue:**
```sql
DELETE FROM media.processing_queue 
WHERE status = 'completed' 
AND completed_at < NOW() - INTERVAL '7 days';
```

**Old audit logs (archive to cold storage):**
```sql
-- Archive to S3/R2, then delete
DELETE FROM audit.record_changes 
WHERE created_at < NOW() - INTERVAL '1 year';
```

### Migration Management

Migrations are located in `/migrations/postgres/` directory.

**Migration naming convention:**
```
NNNNNN_description.up.sql
NNNNNN_description.down.sql
```

**Running migrations:**
```bash
# Using golang-migrate
migrate -path ./migrations/postgres -database "postgresql://..." up

# Rollback one migration
migrate -path ./migrations/postgres -database "postgresql://..." down 1
```

### Troubleshooting

#### Common Issues

**High CPU usage:**
- Check for missing indexes with `pg_stat_user_tables`
- Identify expensive queries with `pg_stat_statements`
- Review autovacuum settings

**Slow queries:**
- Use `EXPLAIN ANALYZE` to understand query plans
- Check index usage with `pg_stat_user_indexes`
- Review WHERE clause selectivity

**Connection exhaustion:**
- Check for connection leaks in application
- Review connection pool settings
- Increase `max_connections` if necessary

**Replication lag:**
- Check network latency
- Review WAL generation rate
- Ensure replica has sufficient resources

---

## Appendix

### Schema ERD

```
auth.users (1) ──< (N) auth.sessions
     │
     ├──< (1) users.profiles
     │
     ├──< (N) users.contacts
     │
     ├──< (N) messages.conversation_participants >── (1) messages.conversations
     │                                                         │
     ├──< (N) messages.messages ────────────────────────────<─┘
     │
     ├──< (N) media.files
     │
     ├──< (N) notifications.notifications
     │
     └──< (N) analytics.events
```

### Extension Requirements

```sql
-- Required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";      -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";       -- Encryption functions
CREATE EXTENSION IF NOT EXISTS "pg_trgm";        -- Trigram text search
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements"; -- Query statistics
```

### Environment Variables

```bash
# Database connection
DATABASE_URL=postgresql://user:password@host:5432/echo_backend
DATABASE_MAX_CONNECTIONS=100
DATABASE_CONNECTION_TIMEOUT=30s

# Migration
MIGRATION_PATH=./migrations/postgres

# Backup
BACKUP_S3_BUCKET=echo-backups
BACKUP_RETENTION_DAYS=30
```

### Performance Baselines

**Target Performance:**
- Simple SELECT: <5ms
- JOIN queries (2-3 tables): <50ms
- Full-text search: <100ms
- Aggregation queries: <200ms
- INSERT/UPDATE: <10ms

**Scalability Targets:**
- 1M+ users
- 100M+ messages
- 10M+ daily active users
- 1000+ requests/second per service

---

## Changelog

### Version 1.0 (November 2025)
- Initial database schema design
- All 7 core schemas implemented
- Row-level security policies
- Comprehensive triggers and functions
- Performance indexes
- Audit trail system
- Analytics views

---

## Support & Contact

For database-related questions or issues:

- **Documentation:** See `/docs/` directory
- **Migrations:** See `/migrations/postgres/` directory
- **Issues:** Create issue in project repository
- **DBA Team:** [Contact information]

---

**Last Updated:** November 3, 2025  
**Schema Version:** 1.0  
**PostgreSQL Version:** 14+
