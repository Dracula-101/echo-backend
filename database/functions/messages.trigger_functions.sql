-- =====================================================
-- MESSAGES SCHEMA — TRIGGER HANDLER FUNCTIONS
-- =====================================================
--
-- Description:  Functions used exclusively as trigger handlers
--               for tables in the messages schema.
--
-- Dependencies: messages schema tables must exist.
--               users.is_blocked() utility function
--               (defined in users.functions.sql).
--
-- Execution:    Load AFTER messages.functions.sql and
--               users.functions.sql, and BEFORE any
--               messages trigger files.
--
-- =====================================================


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 1: TIMESTAMP MANAGEMENT
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_updated_at_column()
-- -------------------------------------------------
-- Automatically sets the `updated_at` column to the
-- current timestamp whenever a row is modified.
-- Used by: messages.conversations,
--          messages.conversation_participants,
--          messages.messages, messages.message_reports,
--          messages.drafts
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 2: CONVERSATION LIFECYCLE
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.create_default_conversation_settings()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.conversations.
-- Creates a default conversation_settings row with
-- all settings at their default values.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.create_default_conversation_settings()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.conversation_settings (conversation_id)
    VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.add_creator_as_participant()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.conversations.
-- Automatically adds the conversation creator as the
-- first participant. Role is 'owner' for groups/channels,
-- 'member' for direct conversations.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.add_creator_as_participant()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.conversation_participants (
        conversation_id, user_id, role, joined_at
    ) VALUES (
        NEW.id,
        NEW.creator_user_id,
        CASE
            WHEN NEW.conversation_type = 'direct' THEN 'member'
            ELSE 'owner'
        END,
        NOW()
    );

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.update_member_count()
-- -------------------------------------------------
-- Fires AFTER INSERT, UPDATE, or DELETE on
-- messages.conversation_participants. Recalculates
-- the member_count on the parent conversation by
-- counting active (not left, not removed) participants.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_member_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversations
    SET member_count = (
        SELECT COUNT(*)
        FROM messages.conversation_participants
        WHERE conversation_id = COALESCE(NEW.conversation_id, OLD.conversation_id)
          AND left_at IS NULL
          AND removed_at IS NULL
    )
    WHERE id = COALESCE(NEW.conversation_id, OLD.conversation_id);

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 3: MESSAGE LIFECYCLE
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_conversation_last_message()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages. Updates
-- the parent conversation's last_message_id,
-- last_message_at, last_activity_at, and increments
-- the message_count.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_conversation_last_message()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversations
    SET last_message_id = NEW.id,
        last_message_at = NEW.created_at,
        last_activity_at = NEW.created_at,
        message_count = message_count + 1
    WHERE id = NEW.conversation_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.set_edited_timestamp()
-- -------------------------------------------------
-- Fires BEFORE UPDATE on messages.messages when the
-- content column changes. Marks the message as edited,
-- records the edit timestamp, and appends the previous
-- content to the edit_history JSONB array.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.set_edited_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.content IS DISTINCT FROM NEW.content AND NEW.content IS NOT NULL THEN
        NEW.is_edited = TRUE;
        NEW.edited_at = NOW();

        -- Append previous content to edit history
        NEW.edit_history = COALESCE(NEW.edit_history, '[]'::JSONB) ||
            jsonb_build_object(
                'edited_at', NOW(),
                'previous_content', OLD.content
            );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.set_message_expiration()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.messages. If the
-- conversation has disappearing messages enabled,
-- calculates and sets the expires_at timestamp based
-- on the configured duration.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.set_message_expiration()
RETURNS TRIGGER AS $$
DECLARE
    v_expire_after INTEGER;
BEGIN
    SELECT cs.disappearing_messages_duration INTO v_expire_after
    FROM messages.conversation_settings cs
    WHERE cs.conversation_id = NEW.conversation_id
      AND cs.disappearing_messages_enabled = TRUE;

    IF v_expire_after IS NOT NULL THEN
        NEW.expire_after_seconds = v_expire_after;
        NEW.expires_at = NOW() + (v_expire_after || ' seconds')::INTERVAL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.update_reply_count()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages when
-- parent_message_id is set. Increments reply_count
-- and updates last_reply_at on the parent message
-- to support threaded conversations.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_reply_count()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.parent_message_id IS NOT NULL THEN
        UPDATE messages.messages
        SET reply_count = reply_count + 1,
            last_reply_at = NEW.created_at
        WHERE id = NEW.parent_message_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.increment_forward_count()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages when
-- forwarded_from_message_id is set. Increments the
-- forward_count on the original message to track
-- message virality.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.increment_forward_count()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.forwarded_from_message_id IS NOT NULL THEN
        UPDATE messages.messages
        SET forward_count = forward_count + 1
        WHERE id = NEW.forwarded_from_message_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 4: DELIVERY TRACKING
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.create_delivery_status()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages. Creates
-- a 'sent' delivery_status row for every active
-- participant in the conversation (except the sender).
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.create_delivery_status()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.delivery_status (message_id, user_id, status)
    SELECT NEW.id, cp.user_id, 'sent'
    FROM messages.conversation_participants cp
    WHERE cp.conversation_id = NEW.conversation_id
      AND cp.user_id != NEW.sender_user_id
      AND cp.left_at IS NULL
      AND cp.removed_at IS NULL;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.update_message_delivery_counts()
-- -------------------------------------------------
-- Fires AFTER INSERT or UPDATE on
-- messages.delivery_status. Recalculates the
-- delivery_count and read_count on the parent
-- message based on current delivery_status rows.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_message_delivery_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND OLD.status IS DISTINCT FROM NEW.status) THEN
        UPDATE messages.messages
        SET delivery_count = (
            SELECT COUNT(*) FROM messages.delivery_status
            WHERE message_id = COALESCE(NEW.message_id, OLD.message_id)
              AND status IN ('delivered', 'read')
        ),
        read_count = (
            SELECT COUNT(*) FROM messages.delivery_status
            WHERE message_id = COALESCE(NEW.message_id, OLD.message_id)
              AND status = 'read'
        )
        WHERE id = COALESCE(NEW.message_id, OLD.message_id);
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.increment_unread_count()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages. Increments
-- the unread_count for all active participants in the
-- conversation except the sender.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.increment_unread_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversation_participants
    SET unread_count = unread_count + 1
    WHERE conversation_id = NEW.conversation_id
      AND user_id != NEW.sender_user_id
      AND left_at IS NULL
      AND removed_at IS NULL;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.increment_mention_count()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages when the
-- mentions JSONB array is non-empty. Increments
-- mention_count for each mentioned user's participant
-- record in the conversation.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.increment_mention_count()
RETURNS TRIGGER AS $$
DECLARE
    v_mentioned_user_id UUID;
BEGIN
    IF NEW.mentions IS NOT NULL AND jsonb_array_length(NEW.mentions) > 0 THEN
        FOR v_mentioned_user_id IN
            SELECT (mention->>'user_id')::UUID
            FROM jsonb_array_elements(NEW.mentions) AS mention
        LOOP
            UPDATE messages.conversation_participants
            SET mention_count = mention_count + 1
            WHERE conversation_id = NEW.conversation_id
              AND user_id = v_mentioned_user_id;
        END LOOP;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 5: SEARCH INDEX
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_search_index()
-- -------------------------------------------------
-- Fires AFTER INSERT or UPDATE on messages.messages
-- for text messages with non-null content that are
-- not deleted. Upserts a tsvector into the
-- messages.search_index table for full-text search.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_search_index()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.search_index (
        message_id, conversation_id, user_id, content_tsvector
    ) VALUES (
        NEW.id, NEW.conversation_id, NEW.sender_user_id,
        to_tsvector('english', COALESCE(NEW.content, ''))
    )
    ON CONFLICT (message_id) DO UPDATE SET
        content_tsvector = to_tsvector('english', COALESCE(NEW.content, '')),
        updated_at = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 6: REACTIONS
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_reaction_count()
-- -------------------------------------------------
-- Fires AFTER INSERT or DELETE on messages.reactions.
-- Recalculates the reaction_count on the parent
-- message by counting all reaction rows.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_reaction_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.messages
    SET reaction_count = (
        SELECT COUNT(*) FROM messages.reactions
        WHERE message_id = COALESCE(NEW.message_id, OLD.message_id)
    )
    WHERE id = COALESCE(NEW.message_id, OLD.message_id);

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 7: POLLS
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_poll_votes()
-- -------------------------------------------------
-- Fires AFTER INSERT or DELETE on messages.poll_votes.
-- Recalculates vote counts and percentages for the
-- affected poll option and the poll overall.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_poll_votes()
RETURNS TRIGGER AS $$
BEGIN
    -- Update the specific option's vote count
    UPDATE messages.poll_options
    SET vote_count = (
        SELECT COUNT(*) FROM messages.poll_votes
        WHERE poll_option_id = NEW.poll_option_id
    )
    WHERE id = NEW.poll_option_id;

    -- Update the poll's total vote count
    UPDATE messages.polls
    SET total_votes = (
        SELECT COUNT(*) FROM messages.poll_votes
        WHERE poll_id = NEW.poll_id
    )
    WHERE id = NEW.poll_id;

    -- Recalculate all option percentages for this poll
    UPDATE messages.poll_options po
    SET vote_percentage = CASE
        WHEN p.total_votes > 0 THEN
            (po.vote_count::DECIMAL / p.total_votes * 100)
        ELSE 0
    END
    FROM messages.polls p
    WHERE po.poll_id = p.id
      AND p.id = NEW.poll_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.auto_close_poll()
-- -------------------------------------------------
-- Fires BEFORE UPDATE on messages.polls when
-- closes_at has passed and the poll is still open.
-- Automatically marks the poll as closed.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.auto_close_poll()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.closes_at IS NOT NULL AND NEW.closes_at <= NOW() AND NOT NEW.is_closed THEN
        NEW.is_closed = TRUE;
        NEW.closed_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.validate_poll_not_closed()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.poll_votes.
-- Prevents voting on a poll that has already been
-- closed, raising an exception if attempted.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.validate_poll_not_closed()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM messages.polls
        WHERE id = NEW.poll_id
          AND is_closed = TRUE
    ) THEN
        RAISE EXCEPTION 'Cannot vote on closed poll';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.validate_single_vote()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.poll_votes.
-- For polls that do not allow multiple answers,
-- prevents a user from casting more than one vote.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.validate_single_vote()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM messages.polls
        WHERE id = NEW.poll_id
          AND allow_multiple_answers = TRUE
    ) THEN
        IF EXISTS (
            SELECT 1 FROM messages.poll_votes
            WHERE poll_id = NEW.poll_id
              AND user_id = NEW.user_id
        ) THEN
            RAISE EXCEPTION 'User has already voted on this poll';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 8: VALIDATION & CLEANUP
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.validate_participant_can_send()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.messages. Ensures
-- the sender is an active participant in the
-- conversation with send permissions. Raises an
-- exception if the check fails.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.validate_participant_can_send()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = NEW.conversation_id
          AND user_id = NEW.sender_user_id
          AND left_at IS NULL
          AND removed_at IS NULL
          AND can_send_messages = TRUE
    ) THEN
        RAISE EXCEPTION 'User % cannot send messages in conversation %',
            NEW.sender_user_id, NEW.conversation_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.validate_not_blocked()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.messages. For
-- direct conversations, checks whether either party
-- has blocked the other. Raises an exception if a
-- block exists in either direction.
--
-- Depends on: users.is_blocked() from users.functions.sql
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.validate_not_blocked()
RETURNS TRIGGER AS $$
DECLARE
    v_recipient_id UUID;
BEGIN
    -- Only enforce block checks on direct conversations
    IF EXISTS (
        SELECT 1 FROM messages.conversations c
        WHERE c.id = NEW.conversation_id
          AND c.conversation_type = 'direct'
    ) THEN
        -- Find the other participant
        SELECT user_id INTO v_recipient_id
        FROM messages.conversation_participants
        WHERE conversation_id = NEW.conversation_id
          AND user_id != NEW.sender_user_id
        LIMIT 1;

        IF v_recipient_id IS NOT NULL THEN
            IF users.is_blocked(NEW.sender_user_id, v_recipient_id)
               OR users.is_blocked(v_recipient_id, NEW.sender_user_id) THEN
                RAISE EXCEPTION 'Cannot send message to blocked user';
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- messages.cleanup_typing_indicator()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.typing_indicators.
-- Removes all expired typing indicators to keep the
-- table lean. Expired indicators are those past their
-- expires_at timestamp.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.cleanup_typing_indicator()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM messages.typing_indicators
    WHERE expires_at < NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
