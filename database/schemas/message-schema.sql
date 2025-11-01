-- =====================================================
-- MESSAGE SCHEMA - Conversations & Messages
-- =====================================================

-- Create Schema
CREATE SCHEMA IF NOT EXISTS messages;

-- Conversations/Chats
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
    
    -- Group/Channel specific
    max_members INTEGER,
    is_public BOOLEAN DEFAULT FALSE,
    invite_link TEXT UNIQUE,
    invite_link_expires_at TIMESTAMPTZ,
    join_approval_required BOOLEAN DEFAULT FALSE,
    
    -- Permissions
    who_can_send_messages VARCHAR(50) DEFAULT 'all', -- all, admins, specific_roles
    who_can_add_members VARCHAR(50) DEFAULT 'admins',
    who_can_edit_info VARCHAR(50) DEFAULT 'admins',
    who_can_pin_messages VARCHAR(50) DEFAULT 'admins',
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    is_archived BOOLEAN DEFAULT FALSE,
    archived_at TIMESTAMPTZ,
    
    -- Stats
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

-- Conversation Participants
CREATE TABLE messages.conversation_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'member', -- owner, admin, moderator, member
    
    -- Participant specific settings
    nickname VARCHAR(100),
    custom_notifications BOOLEAN,
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    is_pinned BOOLEAN DEFAULT FALSE,
    pin_order INTEGER,
    is_archived BOOLEAN DEFAULT FALSE,
    
    -- Message tracking
    last_read_message_id UUID,
    last_read_at TIMESTAMPTZ,
    unread_count INTEGER DEFAULT 0,
    mention_count INTEGER DEFAULT 0,
    
    -- Permissions
    can_send_messages BOOLEAN DEFAULT TRUE,
    can_send_media BOOLEAN DEFAULT TRUE,
    can_add_members BOOLEAN DEFAULT FALSE,
    can_remove_members BOOLEAN DEFAULT FALSE,
    can_edit_info BOOLEAN DEFAULT FALSE,
    can_pin_messages BOOLEAN DEFAULT FALSE,
    can_delete_messages BOOLEAN DEFAULT FALSE,
    
    -- Status
    join_method VARCHAR(50), -- invited, link, added, search
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

-- Messages
CREATE TABLE messages.messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    sender_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE SET NULL,
    parent_message_id UUID REFERENCES messages.messages(id) ON DELETE SET NULL, -- For replies/threads
    
    -- Content
    message_type VARCHAR(50) DEFAULT 'text', -- text, image, video, audio, document, location, contact, sticker, gif, poll
    content TEXT,
    content_encrypted BOOLEAN DEFAULT TRUE,
    content_hash TEXT, -- For deduplication
    
    -- Formatting
    format_type VARCHAR(50) DEFAULT 'plain', -- plain, markdown, html
    mentions JSONB DEFAULT '[]'::JSONB, -- [{user_id, offset, length}]
    hashtags TEXT[],
    links JSONB DEFAULT '[]'::JSONB, -- [{url, title, preview}]
    
    -- Status
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
    
    -- Delivery & Read tracking
    delivered_at TIMESTAMPTZ,
    delivery_count INTEGER DEFAULT 0,
    read_count INTEGER DEFAULT 0,
    
    -- Moderation
    is_flagged BOOLEAN DEFAULT FALSE,
    flag_reason TEXT,
    flagged_at TIMESTAMPTZ,
    flagged_by_user_id UUID REFERENCES auth.users(id),
    
    -- Scheduling
    scheduled_at TIMESTAMPTZ,
    is_scheduled BOOLEAN DEFAULT FALSE,
    
    -- Reply/Thread stats
    reply_count INTEGER DEFAULT 0,
    last_reply_at TIMESTAMPTZ,
    
    -- Reactions
    reaction_count INTEGER DEFAULT 0,
    
    -- Forwarding
    is_forwarded BOOLEAN DEFAULT FALSE,
    forwarded_from_message_id UUID REFERENCES messages.messages(id),
    forward_count INTEGER DEFAULT 0,
    
    -- Device info
    sent_from_device_id VARCHAR(255),
    sent_from_ip INET,
    
    -- Expiration (for disappearing messages)
    expires_at TIMESTAMPTZ,
    expire_after_seconds INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Message Reactions
CREATE TABLE messages.reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    reaction_type VARCHAR(100) NOT NULL, -- emoji or custom reaction code
    reaction_emoji VARCHAR(100),
    reaction_skin_tone VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id, reaction_type)
);

-- Message Delivery Status (per recipient)
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

-- Message Attachments/Media (references media schema)
CREATE TABLE messages.message_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    media_id UUID NOT NULL, -- References media.files
    media_type VARCHAR(50) NOT NULL, -- image, video, audio, document, voice, sticker
    display_order INTEGER DEFAULT 0,
    caption TEXT,
    thumbnail_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Message Links Preview
CREATE TABLE messages.link_previews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    title TEXT,
    description TEXT,
    image_url TEXT,
    favicon_url TEXT,
    site_name VARCHAR(255),
    content_type VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, url)
);

-- Polls
CREATE TABLE messages.polls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID UNIQUE NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    allow_multiple_answers BOOLEAN DEFAULT FALSE,
    is_anonymous BOOLEAN DEFAULT FALSE,
    is_quiz BOOLEAN DEFAULT FALSE,
    correct_option_id INTEGER,
    explanation TEXT,
    closes_at TIMESTAMPTZ,
    is_closed BOOLEAN DEFAULT FALSE,
    closed_at TIMESTAMPTZ,
    total_votes INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Poll Options
CREATE TABLE messages.poll_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES messages.polls(id) ON DELETE CASCADE,
    option_text TEXT NOT NULL,
    option_order INTEGER NOT NULL,
    vote_count INTEGER DEFAULT 0,
    vote_percentage DECIMAL(5,2) DEFAULT 0.00,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Poll Votes
CREATE TABLE messages.poll_votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES messages.polls(id) ON DELETE CASCADE,
    poll_option_id UUID NOT NULL REFERENCES messages.poll_options(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    voted_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(poll_id, poll_option_id, user_id)
);

-- Typing Indicators
CREATE TABLE messages.typing_indicators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL, -- Typically 5-10 seconds from started_at
    UNIQUE(conversation_id, user_id)
);

-- Message Reports
CREATE TABLE messages.message_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    reporter_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    report_type VARCHAR(100) NOT NULL, -- spam, harassment, violence, inappropriate, copyright
    report_category VARCHAR(100),
    description TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    priority VARCHAR(20) DEFAULT 'medium',
    assigned_to UUID REFERENCES auth.users(id),
    resolution TEXT,
    action_taken VARCHAR(100), -- none, warning, message_deleted, user_banned
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Message Drafts
CREATE TABLE messages.drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    content TEXT,
    reply_to_message_id UUID REFERENCES messages.messages(id),
    mentions JSONB DEFAULT '[]'::JSONB,
    attachments JSONB DEFAULT '[]'::JSONB, -- Temporary attachment info
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, conversation_id)
);

-- Message Bookmarks/Saved Messages
CREATE TABLE messages.bookmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    collection_name VARCHAR(255), -- Optional organization
    notes TEXT,
    tags TEXT[],
    bookmarked_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, message_id)
);

-- Pinned Messages (for conversations)
CREATE TABLE messages.pinned_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    pinned_by_user_id UUID NOT NULL REFERENCES auth.users(id),
    pin_order INTEGER DEFAULT 0,
    pinned_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(conversation_id, message_id)
);

-- Conversation Invites
CREATE TABLE messages.conversation_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    inviter_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    invitee_user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    invitee_phone_number VARCHAR(20),
    invitee_email VARCHAR(255),
    invite_code TEXT UNIQUE,
    status VARCHAR(50) DEFAULT 'pending', -- pending, accepted, rejected, expired, revoked
    max_uses INTEGER DEFAULT 1,
    use_count INTEGER DEFAULT 0,
    expires_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Message Search Index (for full-text search optimization)
CREATE TABLE messages.search_index (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID UNIQUE NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id),
    content_tsvector TSVECTOR,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Voice/Video Calls
CREATE TABLE messages.calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    call_type VARCHAR(50) NOT NULL, -- voice, video
    initiator_user_id UUID NOT NULL REFERENCES auth.users(id),
    status VARCHAR(50) DEFAULT 'initiated', -- initiated, ringing, active, ended, missed, rejected, failed
    
    -- Call details
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    
    -- Quality metrics
    video_quality VARCHAR(50),
    audio_quality VARCHAR(50),
    connection_quality VARCHAR(50),
    packet_loss_percentage DECIMAL(5,2),
    
    -- Server details
    media_server_id VARCHAR(255),
    room_id VARCHAR(255),
    
    end_reason VARCHAR(100), -- completed, missed, rejected, failed, busy, network_error
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Call Participants
CREATE TABLE messages.call_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id UUID NOT NULL REFERENCES messages.calls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'invited', -- invited, ringing, joined, left, rejected
    joined_at TIMESTAMPTZ,
    left_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    is_video_enabled BOOLEAN DEFAULT FALSE,
    is_audio_enabled BOOLEAN DEFAULT TRUE,
    is_screen_sharing BOOLEAN DEFAULT FALSE,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(call_id, user_id)
);

-- Conversation Settings
CREATE TABLE messages.conversation_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID UNIQUE NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    disappearing_messages_enabled BOOLEAN DEFAULT FALSE,
    disappearing_messages_duration INTEGER, -- seconds
    message_history_enabled BOOLEAN DEFAULT TRUE,
    screenshot_notification BOOLEAN DEFAULT FALSE,
    read_receipts_enabled BOOLEAN DEFAULT TRUE,
    typing_indicators_enabled BOOLEAN DEFAULT TRUE,
    link_previews_enabled BOOLEAN DEFAULT TRUE,
    auto_download_media BOOLEAN DEFAULT TRUE,
    message_request_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_conversations_creator ON messages.conversations(creator_user_id);
CREATE INDEX idx_conversations_last_message ON messages.conversations(last_message_at);
CREATE INDEX idx_participants_conversation ON messages.conversation_participants(conversation_id);
CREATE INDEX idx_participants_user ON messages.conversation_participants(user_id);
CREATE INDEX idx_messages_conversation ON messages.messages(conversation_id);
CREATE INDEX idx_messages_sender ON messages.messages(sender_user_id);
CREATE INDEX idx_messages_created ON messages.messages(created_at);
CREATE INDEX idx_messages_parent ON messages.messages(parent_message_id);
CREATE INDEX idx_reactions_message ON messages.reactions(message_id);
CREATE INDEX idx_reactions_user ON messages.reactions(user_id);
CREATE INDEX idx_delivery_status_message ON messages.delivery_status(message_id);
CREATE INDEX idx_delivery_status_user ON messages.delivery_status(user_id);
CREATE INDEX idx_search_index_conversation ON messages.search_index(conversation_id);
CREATE INDEX idx_search_index_content ON messages.search_index USING GIN(content_tsvector);