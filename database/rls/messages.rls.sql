-- =====================================================
-- MESSAGES SCHEMA - ROW LEVEL SECURITY (RLS)
-- =====================================================

-- Enable RLS on all messages tables
ALTER TABLE messages.conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.conversation_participants ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.reactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.delivery_status ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.message_media ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.link_previews ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.polls ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.poll_options ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.poll_votes ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.typing_indicators ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.message_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.drafts ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.bookmarks ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.pinned_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.conversation_invites ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.search_index ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.calls ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.call_participants ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages.conversation_settings ENABLE ROW LEVEL SECURITY;

-- =====================================================
-- CONVERSATIONS TABLE POLICIES
-- =====================================================

-- Users can view conversations they are participants in
CREATE POLICY conversations_select_participant
    ON messages.conversations
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.conversations.id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
            AND cp.removed_at IS NULL
        )
    );

-- Users can view public conversations
CREATE POLICY conversations_select_public
    ON messages.conversations
    FOR SELECT
    USING (is_public = TRUE AND is_active = TRUE);

-- Users can create conversations
CREATE POLICY conversations_insert_own
    ON messages.conversations
    FOR INSERT
    WITH CHECK (creator_user_id = auth.current_user_id());

-- Conversation owners/admins can update
CREATE POLICY conversations_update_owner_admin
    ON messages.conversations
    FOR UPDATE
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.conversations.id
            AND cp.user_id = auth.current_user_id()
            AND cp.role IN ('owner', 'admin')
            AND cp.can_edit_info = TRUE
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.conversations.id
            AND cp.user_id = auth.current_user_id()
            AND cp.role IN ('owner', 'admin')
        )
    );

-- Admins can manage all conversations
CREATE POLICY conversations_admin_all
    ON messages.conversations
    FOR ALL
    USING (auth.is_admin());

-- =====================================================
-- CONVERSATION PARTICIPANTS POLICIES
-- =====================================================

-- Users can view participants in conversations they're part of
CREATE POLICY participants_select_same_conversation
    ON messages.conversation_participants
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.conversation_participants.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
            AND cp.removed_at IS NULL
        )
        OR EXISTS (
            SELECT 1 FROM messages.conversations c
            WHERE c.id = messages.conversation_participants.conversation_id
            AND c.is_public = TRUE
        )
    );

-- Admins can add participants
CREATE POLICY participants_insert_admin
    ON messages.conversation_participants
    FOR INSERT
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.conversation_participants.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.can_add_members = TRUE
        )
        OR user_id = auth.current_user_id() -- Can add self via invite
    );

-- Users can update their own participant settings
CREATE POLICY participants_update_own
    ON messages.conversation_participants
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (
        user_id = auth.current_user_id()
    );

-- Admins can update other participants
CREATE POLICY participants_update_admin
    ON messages.conversation_participants
    FOR UPDATE
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.conversation_participants.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.role IN ('owner', 'admin')
        )
    );

-- Users can leave conversations (delete self)
CREATE POLICY participants_delete_self
    ON messages.conversation_participants
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Admins can remove participants
CREATE POLICY participants_delete_admin
    ON messages.conversation_participants
    FOR DELETE
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.conversation_participants.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.can_remove_members = TRUE
        )
    );

-- =====================================================
-- MESSAGES TABLE POLICIES
-- =====================================================

-- Users can view messages in their conversations
CREATE POLICY messages_select_participant
    ON messages.messages
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.messages.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
            AND cp.removed_at IS NULL
        )
        AND NOT EXISTS (
            SELECT 1 FROM users.blocked_users b
            WHERE (
                (b.user_id = auth.current_user_id() AND b.blocked_user_id = messages.messages.sender_user_id)
                OR (b.user_id = messages.messages.sender_user_id AND b.blocked_user_id = auth.current_user_id())
            )
            AND b.unblocked_at IS NULL
        )
    );

-- Users can insert messages in conversations they're part of
CREATE POLICY messages_insert_participant
    ON messages.messages
    FOR INSERT
    WITH CHECK (
        sender_user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.messages.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.can_send_messages = TRUE
            AND cp.left_at IS NULL
            AND cp.removed_at IS NULL
        )
    );

-- Users can update their own messages
CREATE POLICY messages_update_own
    ON messages.messages
    FOR UPDATE
    USING (sender_user_id = auth.current_user_id())
    WITH CHECK (sender_user_id = auth.current_user_id());

-- Users can delete their own messages
CREATE POLICY messages_delete_own
    ON messages.messages
    FOR DELETE
    USING (
        sender_user_id = auth.current_user_id()
        OR EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.messages.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.can_delete_messages = TRUE
        )
    );

-- =====================================================
-- REACTIONS TABLE POLICIES
-- =====================================================

-- Users can view reactions in conversations they're part of
CREATE POLICY reactions_select_participant
    ON messages.reactions
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.messages m
            JOIN messages.conversation_participants cp ON cp.conversation_id = m.conversation_id
            WHERE m.id = messages.reactions.message_id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
        )
    );

-- Users can add reactions
CREATE POLICY reactions_insert_participant
    ON messages.reactions
    FOR INSERT
    WITH CHECK (
        user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM messages.messages m
            JOIN messages.conversation_participants cp ON cp.conversation_id = m.conversation_id
            WHERE m.id = messages.reactions.message_id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
        )
    );

-- Users can delete their own reactions
CREATE POLICY reactions_delete_own
    ON messages.reactions
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- =====================================================
-- DELIVERY STATUS POLICIES
-- =====================================================

-- Users can view delivery status for their messages
CREATE POLICY delivery_status_select_sender
    ON messages.delivery_status
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.messages m
            WHERE m.id = messages.delivery_status.message_id
            AND m.sender_user_id = auth.current_user_id()
        )
    );

-- Users can view their own delivery status
CREATE POLICY delivery_status_select_own
    ON messages.delivery_status
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Service role can manage delivery status
CREATE POLICY delivery_status_service_all
    ON messages.delivery_status
    FOR ALL
    USING (TRUE);

-- Users can update their own delivery status
CREATE POLICY delivery_status_update_own
    ON messages.delivery_status
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- MESSAGE MEDIA POLICIES
-- =====================================================

-- Users can view media in conversations they're part of
CREATE POLICY message_media_select_participant
    ON messages.message_media
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.messages m
            JOIN messages.conversation_participants cp ON cp.conversation_id = m.conversation_id
            WHERE m.id = messages.message_media.message_id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
        )
    );

-- Service role can manage message media
CREATE POLICY message_media_service_all
    ON messages.message_media
    FOR ALL
    USING (TRUE);

-- =====================================================
-- POLLS POLICIES
-- =====================================================

-- Users can view polls in conversations they're part of
CREATE POLICY polls_select_participant
    ON messages.polls
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.messages m
            JOIN messages.conversation_participants cp ON cp.conversation_id = m.conversation_id
            WHERE m.id = messages.polls.message_id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
        )
    );

-- Service role can manage polls
CREATE POLICY polls_service_all
    ON messages.polls
    FOR ALL
    USING (TRUE);

-- =====================================================
-- POLL OPTIONS POLICIES
-- =====================================================

-- Users can view poll options
CREATE POLICY poll_options_select
    ON messages.poll_options
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.polls p
            JOIN messages.messages m ON m.id = p.message_id
            JOIN messages.conversation_participants cp ON cp.conversation_id = m.conversation_id
            WHERE p.id = messages.poll_options.poll_id
            AND cp.user_id = auth.current_user_id()
        )
    );

-- Service role can manage poll options
CREATE POLICY poll_options_service_all
    ON messages.poll_options
    FOR ALL
    USING (TRUE);

-- =====================================================
-- POLL VOTES POLICIES
-- =====================================================

-- Users can view votes (if poll is not anonymous)
CREATE POLICY poll_votes_select_not_anonymous
    ON messages.poll_votes
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.polls p
            WHERE p.id = messages.poll_votes.poll_id
            AND p.is_anonymous = FALSE
        )
        AND EXISTS (
            SELECT 1 FROM messages.polls p
            JOIN messages.messages m ON m.id = p.message_id
            JOIN messages.conversation_participants cp ON cp.conversation_id = m.conversation_id
            WHERE p.id = messages.poll_votes.poll_id
            AND cp.user_id = auth.current_user_id()
        )
    );

-- Users can view their own votes
CREATE POLICY poll_votes_select_own
    ON messages.poll_votes
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can vote
CREATE POLICY poll_votes_insert_participant
    ON messages.poll_votes
    FOR INSERT
    WITH CHECK (
        user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM messages.polls p
            JOIN messages.messages m ON m.id = p.message_id
            JOIN messages.conversation_participants cp ON cp.conversation_id = m.conversation_id
            WHERE p.id = messages.poll_votes.poll_id
            AND cp.user_id = auth.current_user_id()
        )
    );

-- Users can delete their own votes
CREATE POLICY poll_votes_delete_own
    ON messages.poll_votes
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- =====================================================
-- TYPING INDICATORS POLICIES
-- =====================================================

-- Users can view typing indicators in their conversations
CREATE POLICY typing_indicators_select_participant
    ON messages.typing_indicators
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.typing_indicators.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
        )
    );

-- Users can insert their own typing indicators
CREATE POLICY typing_indicators_insert_own
    ON messages.typing_indicators
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

-- Users can delete their own typing indicators
CREATE POLICY typing_indicators_delete_own
    ON messages.typing_indicators
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- =====================================================
-- DRAFTS POLICIES
-- =====================================================

-- Users can manage their own drafts
CREATE POLICY drafts_all_own
    ON messages.drafts
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- BOOKMARKS POLICIES
-- =====================================================

-- Users can manage their own bookmarks
CREATE POLICY bookmarks_all_own
    ON messages.bookmarks
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- MESSAGE REPORTS POLICIES
-- =====================================================

-- Users can view reports they created
CREATE POLICY message_reports_select_reporter
    ON messages.message_reports
    FOR SELECT
    USING (reporter_user_id = auth.current_user_id());

-- Users can create reports
CREATE POLICY message_reports_insert
    ON messages.message_reports
    FOR INSERT
    WITH CHECK (reporter_user_id = auth.current_user_id());

-- Admins can manage all reports
CREATE POLICY message_reports_admin_all
    ON messages.message_reports
    FOR ALL
    USING (auth.is_admin());

-- =====================================================
-- CALLS POLICIES
-- =====================================================

-- Users can view calls in their conversations
CREATE POLICY calls_select_participant
    ON messages.calls
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.calls.conversation_id
            AND cp.user_id = auth.current_user_id()
        )
    );

-- Users can initiate calls
CREATE POLICY calls_insert_participant
    ON messages.calls
    FOR INSERT
    WITH CHECK (
        initiator_user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.calls.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
        )
    );

-- Service role can manage calls
CREATE POLICY calls_service_all
    ON messages.calls
    FOR ALL
    USING (TRUE);

-- =====================================================
-- CALL PARTICIPANTS POLICIES
-- =====================================================

-- Users can view call participants in their calls
CREATE POLICY call_participants_select
    ON messages.call_participants
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.calls c
            JOIN messages.conversation_participants cp ON cp.conversation_id = c.conversation_id
            WHERE c.id = messages.call_participants.call_id
            AND cp.user_id = auth.current_user_id()
        )
    );

-- Service role can manage call participants
CREATE POLICY call_participants_service_all
    ON messages.call_participants
    FOR ALL
    USING (TRUE);

-- =====================================================
-- SEARCH INDEX POLICIES
-- =====================================================

-- Users can search in their conversations
CREATE POLICY search_index_select_participant
    ON messages.search_index
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.search_index.conversation_id
            AND cp.user_id = auth.current_user_id()
            AND cp.left_at IS NULL
        )
    );

-- Service role can manage search index
CREATE POLICY search_index_service_all
    ON messages.search_index
    FOR ALL
    USING (TRUE);

-- =====================================================
-- OTHER POLICIES
-- =====================================================

-- Link previews, pinned messages, conversation invites, settings
-- follow the same pattern as messages - accessible to conversation participants

CREATE POLICY link_previews_select_participant
    ON messages.link_previews
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.messages m
            JOIN messages.conversation_participants cp ON cp.conversation_id = m.conversation_id
            WHERE m.id = messages.link_previews.message_id
            AND cp.user_id = auth.current_user_id()
        )
    );

CREATE POLICY pinned_messages_select_participant
    ON messages.pinned_messages
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.pinned_messages.conversation_id
            AND cp.user_id = auth.current_user_id()
        )
    );

CREATE POLICY conversation_invites_select_relevant
    ON messages.conversation_invites
    FOR SELECT
    USING (
        inviter_user_id = auth.current_user_id()
        OR invitee_user_id = auth.current_user_id()
        OR EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.conversation_invites.conversation_id
            AND cp.user_id = auth.current_user_id()
        )
    );

CREATE POLICY conversation_settings_select_participant
    ON messages.conversation_settings
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = messages.conversation_settings.conversation_id
            AND cp.user_id = auth.current_user_id()
        )
    );