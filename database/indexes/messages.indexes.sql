-- =====================================================
-- MESSAGE SCHEMA - INDEXES (FIXED)
-- =====================================================

-- Conversations table indexes
CREATE INDEX IF NOT EXISTS idx_messages_conversations_creator ON messages.conversations(creator_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_conversations_type ON messages.conversations(conversation_type);
CREATE INDEX IF NOT EXISTS idx_messages_conversations_last_message ON messages.conversations(last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_conversations_last_activity ON messages.conversations(last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_conversations_active ON messages.conversations(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_conversations_archived ON messages.conversations(is_archived) WHERE is_archived = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_conversations_public ON messages.conversations(is_public) WHERE is_public = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_conversations_invite ON messages.conversations(invite_link) WHERE invite_link IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_conversations_created ON messages.conversations(created_at);

-- Conversation participants table indexes
CREATE INDEX IF NOT EXISTS idx_messages_participants_conversation ON messages.conversation_participants(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_participants_user ON messages.conversation_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_participants_role ON messages.conversation_participants(role);
CREATE INDEX IF NOT EXISTS idx_messages_participants_unread ON messages.conversation_participants(user_id, unread_count) WHERE unread_count > 0;
CREATE INDEX IF NOT EXISTS idx_messages_participants_mentions ON messages.conversation_participants(user_id, mention_count) WHERE mention_count > 0;
CREATE INDEX IF NOT EXISTS idx_messages_participants_muted ON messages.conversation_participants(is_muted) WHERE is_muted = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_participants_pinned ON messages.conversation_participants(user_id, is_pinned, pin_order) WHERE is_pinned = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_participants_archived ON messages.conversation_participants(user_id, is_archived) WHERE is_archived = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_participants_last_read ON messages.conversation_participants(last_read_at);
CREATE INDEX IF NOT EXISTS idx_messages_participants_joined ON messages.conversation_participants(joined_at);
CREATE INDEX IF NOT EXISTS idx_messages_participants_active ON messages.conversation_participants(conversation_id, user_id) 
    WHERE left_at IS NULL AND removed_at IS NULL;

-- Messages table indexes
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages.messages(conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages.messages(sender_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_parent ON messages.messages(parent_message_id) WHERE parent_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_type ON messages.messages(message_type);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages.messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages.messages(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_edited ON messages.messages(is_edited) WHERE is_edited = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_deleted ON messages.messages(is_deleted) WHERE is_deleted = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_pinned ON messages.messages(conversation_id, is_pinned) WHERE is_pinned = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_scheduled ON messages.messages(scheduled_at) WHERE is_scheduled = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_flagged ON messages.messages(is_flagged) WHERE is_flagged = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_expires ON messages.messages(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_hash ON messages.messages(content_hash) WHERE content_hash IS NOT NULL;

-- Reactions table indexes
CREATE INDEX IF NOT EXISTS idx_messages_reactions_message ON messages.reactions(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_reactions_user ON messages.reactions(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_reactions_type ON messages.reactions(reaction_type);
CREATE INDEX IF NOT EXISTS idx_messages_reactions_created ON messages.reactions(created_at);

-- Delivery status table indexes
CREATE INDEX IF NOT EXISTS idx_messages_delivery_message ON messages.delivery_status(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_delivery_user ON messages.delivery_status(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_delivery_status ON messages.delivery_status(status);
CREATE INDEX IF NOT EXISTS idx_messages_delivery_undelivered ON messages.delivery_status(message_id, status) 
    WHERE status IN ('sent', 'failed');
CREATE INDEX IF NOT EXISTS idx_messages_delivery_unread ON messages.delivery_status(user_id, status) WHERE status = 'sent';

-- Message media table indexes
CREATE INDEX IF NOT EXISTS idx_messages_media_message ON messages.message_media(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_media_type ON messages.message_media(media_type);
CREATE INDEX IF NOT EXISTS idx_messages_media_order ON messages.message_media(message_id, display_order);

-- Link previews table indexes
CREATE INDEX IF NOT EXISTS idx_messages_link_previews_message ON messages.link_previews(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_link_previews_url ON messages.link_previews(url);

-- Polls table indexes
CREATE INDEX IF NOT EXISTS idx_messages_polls_message ON messages.polls(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_polls_closed ON messages.polls(is_closed);
CREATE INDEX IF NOT EXISTS idx_messages_polls_closes_at ON messages.polls(closes_at) WHERE closes_at IS NOT NULL;

-- Poll options table indexes
CREATE INDEX IF NOT EXISTS idx_messages_poll_options_poll ON messages.poll_options(poll_id);
CREATE INDEX IF NOT EXISTS idx_messages_poll_options_order ON messages.poll_options(poll_id, option_order);

-- Poll votes table indexes
CREATE INDEX IF NOT EXISTS idx_messages_poll_votes_poll ON messages.poll_votes(poll_id);
CREATE INDEX IF NOT EXISTS idx_messages_poll_votes_option ON messages.poll_votes(poll_option_id);
CREATE INDEX IF NOT EXISTS idx_messages_poll_votes_user ON messages.poll_votes(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_poll_votes_voted_at ON messages.poll_votes(voted_at);

-- Typing indicators table indexes
CREATE INDEX IF NOT EXISTS idx_messages_typing_conversation ON messages.typing_indicators(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_typing_user ON messages.typing_indicators(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_typing_expires ON messages.typing_indicators(expires_at);
-- FIXED: Removed NOW() from index predicate
CREATE INDEX IF NOT EXISTS idx_messages_typing_active ON messages.typing_indicators(conversation_id, expires_at);

-- Message reports table indexes
CREATE INDEX IF NOT EXISTS idx_messages_reports_message ON messages.message_reports(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_reports_reporter ON messages.message_reports(reporter_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_reports_type ON messages.message_reports(report_type);
CREATE INDEX IF NOT EXISTS idx_messages_reports_status ON messages.message_reports(status);
CREATE INDEX IF NOT EXISTS idx_messages_reports_priority ON messages.message_reports(priority);
CREATE INDEX IF NOT EXISTS idx_messages_reports_assigned ON messages.message_reports(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_reports_created ON messages.message_reports(created_at);

-- Drafts table indexes
CREATE INDEX IF NOT EXISTS idx_messages_drafts_user ON messages.drafts(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_drafts_conversation ON messages.drafts(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_drafts_updated ON messages.drafts(updated_at);

-- Bookmarks table indexes
CREATE INDEX IF NOT EXISTS idx_messages_bookmarks_user ON messages.bookmarks(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_bookmarks_message ON messages.bookmarks(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_bookmarks_collection ON messages.bookmarks(user_id, collection_name);
CREATE INDEX IF NOT EXISTS idx_messages_bookmarks_bookmarked_at ON messages.bookmarks(bookmarked_at);

-- Pinned messages table indexes
CREATE INDEX IF NOT EXISTS idx_messages_pinned_conversation ON messages.pinned_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_pinned_message ON messages.pinned_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_pinned_order ON messages.pinned_messages(conversation_id, pin_order);

-- Conversation invites table indexes
CREATE INDEX IF NOT EXISTS idx_messages_invites_conversation ON messages.conversation_invites(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_invites_inviter ON messages.conversation_invites(inviter_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_invites_invitee ON messages.conversation_invites(invitee_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_invites_code ON messages.conversation_invites(invite_code);
CREATE INDEX IF NOT EXISTS idx_messages_invites_status ON messages.conversation_invites(status);
CREATE INDEX IF NOT EXISTS idx_messages_invites_expires ON messages.conversation_invites(expires_at) WHERE expires_at IS NOT NULL;

-- Search index table indexes
CREATE INDEX IF NOT EXISTS idx_messages_search_message ON messages.search_index(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_search_conversation ON messages.search_index(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_search_user ON messages.search_index(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_search_content ON messages.search_index USING GIN(content_tsvector);
CREATE INDEX IF NOT EXISTS idx_messages_search_created ON messages.search_index(created_at);

-- Calls table indexes
CREATE INDEX IF NOT EXISTS idx_messages_calls_conversation ON messages.calls(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_calls_initiator ON messages.calls(initiator_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_calls_type ON messages.calls(call_type);
CREATE INDEX IF NOT EXISTS idx_messages_calls_status ON messages.calls(status);
CREATE INDEX IF NOT EXISTS idx_messages_calls_started ON messages.calls(started_at);
CREATE INDEX IF NOT EXISTS idx_messages_calls_created ON messages.calls(created_at);

-- Call participants table indexes
CREATE INDEX IF NOT EXISTS idx_messages_call_participants_call ON messages.call_participants(call_id);
CREATE INDEX IF NOT EXISTS idx_messages_call_participants_user ON messages.call_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_call_participants_status ON messages.call_participants(status);
CREATE INDEX IF NOT EXISTS idx_messages_call_participants_joined ON messages.call_participants(joined_at);

-- Conversation settings table indexes
CREATE INDEX IF NOT EXISTS idx_messages_settings_conversation ON messages.conversation_settings(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_settings_disappearing ON messages.conversation_settings(disappearing_messages_enabled) 
    WHERE disappearing_messages_enabled = TRUE;