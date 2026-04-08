-- =====================================================
-- MESSAGES SCHEMA — TRIGGER HANDLER FUNCTIONS
-- =====================================================


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 1: TIMESTAMP MANAGEMENT
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_updated_at_column()
-- -------------------------------------------------
-- Automatically sets updated_at to NOW() before any
-- UPDATE on tables that carry an updated_at column.
--
-- Used by: messages.conversations,
--          messages.conversation_participants,
--          messages.messages,
--          messages.message_reports,
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
    VALUES (NEW.id)
    ON CONFLICT (conversation_id) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.add_creator_as_participant()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.conversations.
-- Adds the conversation creator as the first
-- participant.
--
-- Role assignment:
--   'owner'  — group / channel conversations
--   'member' — direct conversations
--
-- Permissions are set by
-- messages.set_participant_permissions() which fires
-- on the subsequent INSERT into
-- conversation_participants.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.add_creator_as_participant()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.conversation_participants (
        conversation_id,
        user_id,
        role,
        join_method,
        joined_at
    ) VALUES (
        NEW.id,
        NEW.creator_user_id,
        CASE
            WHEN NEW.conversation_type IN ('group', 'channel', 'broadcast') THEN 'owner'
            ELSE 'member'
        END,
        'created',
        NOW()
    )
    ON CONFLICT (conversation_id, user_id) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.update_member_count()
-- -------------------------------------------------
-- Fires AFTER INSERT, UPDATE, or DELETE on
-- messages.conversation_participants.
-- Recalculates member_count on the parent conversation
-- by counting active (not left, not removed) rows.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_member_count()
RETURNS TRIGGER AS $$
DECLARE
    v_conversation_id UUID;
BEGIN
    v_conversation_id := COALESCE(NEW.conversation_id, OLD.conversation_id);

    UPDATE messages.conversations
    SET member_count = (
        SELECT COUNT(*)
        FROM messages.conversation_participants
        WHERE conversation_id = v_conversation_id
          AND left_at         IS NULL
          AND removed_at      IS NULL
    )
    WHERE id = v_conversation_id;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 3: PARTICIPANT MANAGEMENT
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.set_participant_permissions()
-- -------------------------------------------------
-- Fires BEFORE INSERT or UPDATE OF role on
-- messages.conversation_participants.
-- Sets all permission columns based on the
-- participant's role so application code never has
-- to manage them manually.
--
-- Permission matrix:
--   owner / admin  — full permissions
--   moderator      — can pin and delete messages,
--                    can add members, cannot remove
--                    members or edit conversation info
--   member         — send only (default)
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.set_participant_permissions()
RETURNS TRIGGER AS $$
BEGIN
    CASE NEW.role
        WHEN 'owner' THEN
            NEW.can_send_messages   := TRUE;
            NEW.can_send_media      := TRUE;
            NEW.can_add_members     := TRUE;
            NEW.can_remove_members  := TRUE;
            NEW.can_edit_info       := TRUE;
            NEW.can_pin_messages    := TRUE;
            NEW.can_delete_messages := TRUE;

        WHEN 'admin' THEN
            NEW.can_send_messages   := TRUE;
            NEW.can_send_media      := TRUE;
            NEW.can_add_members     := TRUE;
            NEW.can_remove_members  := TRUE;
            NEW.can_edit_info       := TRUE;
            NEW.can_pin_messages    := TRUE;
            NEW.can_delete_messages := TRUE;

        WHEN 'moderator' THEN
            NEW.can_send_messages   := TRUE;
            NEW.can_send_media      := TRUE;
            NEW.can_add_members     := TRUE;
            NEW.can_remove_members  := FALSE;
            NEW.can_edit_info       := FALSE;
            NEW.can_pin_messages    := TRUE;
            NEW.can_delete_messages := TRUE;

        ELSE -- 'member' and any unknown role
            NEW.can_send_messages   := TRUE;
            NEW.can_send_media      := TRUE;
            NEW.can_add_members     := FALSE;
            NEW.can_remove_members  := FALSE;
            NEW.can_edit_info       := FALSE;
            NEW.can_pin_messages    := FALSE;
            NEW.can_delete_messages := FALSE;
    END CASE;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 4: MESSAGE LIFECYCLE
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_conversation_last_message()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages.
-- Updates last_message_id, last_message_at,
-- last_activity_at, and increments message_count
-- on the parent conversation.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_conversation_last_message()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversations
    SET last_message_id  = NEW.id,
        last_message_at  = NEW.created_at,
        last_activity_at = NEW.created_at,
        message_count    = message_count + 1
    WHERE id = NEW.conversation_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.set_edited_timestamp()
-- -------------------------------------------------
-- Fires BEFORE UPDATE on messages.messages when the
-- content column changes. Marks the message as
-- edited, records the edit timestamp, and appends
-- the previous content to the edit_history JSONB
-- array for audit purposes.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.set_edited_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.content IS DISTINCT FROM NEW.content
       AND NEW.content IS NOT NULL
       AND NEW.is_deleted = FALSE
    THEN
        NEW.is_edited    = TRUE;
        NEW.edited_at    = NOW();
        NEW.edit_history = COALESCE(NEW.edit_history, '[]'::JSONB) ||
            jsonb_build_object(
                'edited_at',       NOW(),
                'previous_content', OLD.content
            );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.set_message_expiration()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.messages.
-- If the conversation has disappearing messages
-- enabled, calculates and sets expires_at based on
-- the configured duration in conversation_settings.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.set_message_expiration()
RETURNS TRIGGER AS $$
DECLARE
    v_expire_after INTEGER;
BEGIN
    SELECT cs.disappearing_messages_duration
    INTO v_expire_after
    FROM messages.conversation_settings cs
    WHERE cs.conversation_id                = NEW.conversation_id
      AND cs.disappearing_messages_enabled  = TRUE;

    IF FOUND AND v_expire_after IS NOT NULL THEN
        NEW.expire_after_seconds := v_expire_after;
        NEW.expires_at           := NOW() + (v_expire_after || ' seconds')::INTERVAL;
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
        SET reply_count   = reply_count + 1,
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
-- forwarded_from_message_id is set. Increments
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
-- SECTION 5: DELIVERY TRACKING
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.create_delivery_status()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages.
-- Creates a 'sent' delivery_status row for every
-- active participant in the conversation except the
-- sender. Skips participants who have left or been
-- removed.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.create_delivery_status()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.delivery_status (message_id, user_id, status)
    SELECT NEW.id, cp.user_id, 'sent'
    FROM messages.conversation_participants cp
    WHERE cp.conversation_id = NEW.conversation_id
      AND cp.user_id        != NEW.sender_user_id
      AND cp.left_at         IS NULL
      AND cp.removed_at      IS NULL
    ON CONFLICT (message_id, user_id) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.update_message_delivery_counts()
-- -------------------------------------------------
-- Fires AFTER INSERT or UPDATE on
-- messages.delivery_status. Recalculates
-- delivery_count and read_count on the parent
-- message based on current delivery_status rows.
-- Only runs when status actually changes to avoid
-- unnecessary writes.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_message_delivery_counts()
RETURNS TRIGGER AS $$
DECLARE
    v_message_id UUID;
BEGIN
    v_message_id := COALESCE(NEW.message_id, OLD.message_id);

    IF TG_OP = 'INSERT'
       OR (TG_OP = 'UPDATE' AND OLD.status IS DISTINCT FROM NEW.status)
    THEN
        UPDATE messages.messages
        SET delivery_count = (
                SELECT COUNT(*)
                FROM messages.delivery_status
                WHERE message_id = v_message_id
                  AND status IN ('delivered', 'read')
            ),
            read_count = (
                SELECT COUNT(*)
                FROM messages.delivery_status
                WHERE message_id = v_message_id
                  AND status = 'read'
            )
        WHERE id = v_message_id;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.increment_unread_count()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages.
-- Increments unread_count for all active
-- participants in the conversation except the sender.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.increment_unread_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversation_participants
    SET unread_count = unread_count + 1
    WHERE conversation_id = NEW.conversation_id
      AND user_id        != NEW.sender_user_id
      AND left_at         IS NULL
      AND removed_at      IS NULL;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.increment_mention_count()
-- -------------------------------------------------
-- Fires AFTER INSERT on messages.messages when the
-- mentions JSONB array is non-empty. Increments
-- mention_count for each mentioned user's participant
-- record in the conversation. Skips users who are
-- no longer active participants.
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
            WHERE (mention->>'user_id') IS NOT NULL
        LOOP
            UPDATE messages.conversation_participants
            SET mention_count = mention_count + 1
            WHERE conversation_id = NEW.conversation_id
              AND user_id         = v_mentioned_user_id
              AND left_at         IS NULL
              AND removed_at      IS NULL;
        END LOOP;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 6: SEARCH INDEX
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_search_index()
-- -------------------------------------------------
-- Fires AFTER INSERT or UPDATE on messages.messages
-- for text messages with non-null content that are
-- not deleted. Upserts a tsvector into the
-- messages.search_index table for full-text search.
-- Removes the index row when a message is deleted.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_search_index()
RETURNS TRIGGER AS $$
BEGIN
    -- Remove from index if message was deleted
    IF NEW.is_deleted = TRUE THEN
        DELETE FROM messages.search_index WHERE message_id = NEW.id;
        RETURN NEW;
    END IF;

    -- Only index text messages with content
    IF NEW.message_type = 'text' AND NEW.content IS NOT NULL THEN
        INSERT INTO messages.search_index (
            message_id,
            conversation_id,
            user_id,
            content_tsvector
        ) VALUES (
            NEW.id,
            NEW.conversation_id,
            NEW.sender_user_id,
            to_tsvector('english', NEW.content)
        )
        ON CONFLICT (message_id) DO UPDATE
            SET content_tsvector = to_tsvector('english', NEW.content),
                updated_at       = NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 7: REACTIONS
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_reaction_count()
-- -------------------------------------------------
-- Fires AFTER INSERT or DELETE on messages.reactions.
-- Recalculates reaction_count on the parent message
-- by counting all reaction rows.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_reaction_count()
RETURNS TRIGGER AS $$
DECLARE
    v_message_id UUID;
BEGIN
    v_message_id := COALESCE(NEW.message_id, OLD.message_id);

    UPDATE messages.messages
    SET reaction_count = (
        SELECT COUNT(*)
        FROM messages.reactions
        WHERE message_id = v_message_id
    )
    WHERE id = v_message_id;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 8: POLLS
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.update_poll_votes()
-- -------------------------------------------------
-- Fires AFTER INSERT or DELETE on messages.poll_votes.
-- Recalculates vote counts and percentages for the
-- affected poll option and the poll overall.
--
-- FIX: uses COALESCE(NEW, OLD) for both INSERT and
-- DELETE so option/poll IDs are always available.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.update_poll_votes()
RETURNS TRIGGER AS $$
DECLARE
    v_poll_option_id UUID;
    v_poll_id        UUID;
BEGIN
    v_poll_option_id := COALESCE(NEW.poll_option_id, OLD.poll_option_id);
    v_poll_id        := COALESCE(NEW.poll_id,        OLD.poll_id);

    -- Update the specific option's vote count
    UPDATE messages.poll_options
    SET vote_count = (
        SELECT COUNT(*)
        FROM messages.poll_votes
        WHERE poll_option_id = v_poll_option_id
    )
    WHERE id = v_poll_option_id;

    -- Update the poll's total vote count
    UPDATE messages.polls
    SET total_votes = (
        SELECT COUNT(*)
        FROM messages.poll_votes
        WHERE poll_id = v_poll_id
    )
    WHERE id = v_poll_id;

    -- Recalculate all option percentages for this poll
    UPDATE messages.poll_options po
    SET vote_percentage = CASE
        WHEN p.total_votes > 0
            THEN ROUND((po.vote_count::DECIMAL / p.total_votes) * 100, 2)
        ELSE 0
    END
    FROM messages.polls p
    WHERE po.poll_id = p.id
      AND p.id       = v_poll_id;

    RETURN COALESCE(NEW, OLD);
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
    IF NEW.closes_at IS NOT NULL
       AND NEW.closes_at <= NOW()
       AND NEW.is_closed  = FALSE
    THEN
        NEW.is_closed  := TRUE;
        NEW.closed_at  := NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.validate_poll_not_closed()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.poll_votes.
-- Prevents voting on a poll that is already closed.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.validate_poll_not_closed()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM messages.polls
        WHERE id        = NEW.poll_id
          AND is_closed = TRUE
    ) THEN
        RAISE EXCEPTION 'Cannot vote on a closed poll (poll_id: %)', NEW.poll_id;
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
        WHERE id                    = NEW.poll_id
          AND allow_multiple_answers = TRUE
    ) THEN
        IF EXISTS (
            SELECT 1 FROM messages.poll_votes
            WHERE poll_id = NEW.poll_id
              AND user_id = NEW.user_id
        ) THEN
            RAISE EXCEPTION 'User % has already voted on poll %',
                NEW.user_id, NEW.poll_id;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 9: VALIDATION & CLEANUP
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- messages.validate_participant_can_send()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.messages.
-- Ensures the sender is an active participant with
-- can_send_messages = TRUE. Raises an exception
-- otherwise so the INSERT is aborted cleanly.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.validate_participant_can_send()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM messages.conversation_participants
        WHERE conversation_id   = NEW.conversation_id
          AND user_id           = NEW.sender_user_id
          AND left_at           IS NULL
          AND removed_at        IS NULL
          AND can_send_messages = TRUE
    ) THEN
        RAISE EXCEPTION
            'User % is not permitted to send messages in conversation %',
            NEW.sender_user_id, NEW.conversation_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.validate_not_blocked()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.messages.
-- For direct conversations only: checks whether
-- either party has blocked the other. Raises an
-- exception if a block exists in either direction.
--
-- Depends on: users.is_blocked() from users.functions.sql
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.validate_not_blocked()
RETURNS TRIGGER AS $$
DECLARE
    v_recipient_id UUID;
BEGIN
    -- Only enforce block checks on direct conversations
    IF NOT EXISTS (
        SELECT 1 FROM messages.conversations
        WHERE id                = NEW.conversation_id
          AND conversation_type = 'direct'
    ) THEN
        RETURN NEW;
    END IF;

    -- Find the other participant
    SELECT user_id INTO v_recipient_id
    FROM messages.conversation_participants
    WHERE conversation_id = NEW.conversation_id
      AND user_id        != NEW.sender_user_id
      AND left_at         IS NULL
      AND removed_at      IS NULL
    LIMIT 1;

    IF v_recipient_id IS NOT NULL THEN
        IF users.is_blocked(NEW.sender_user_id, v_recipient_id)
           OR users.is_blocked(v_recipient_id, NEW.sender_user_id)
        THEN
            RAISE EXCEPTION 'Cannot send message: a block exists between users % and %',
                NEW.sender_user_id, v_recipient_id;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.cleanup_typing_indicator()
-- -------------------------------------------------
-- Fires BEFORE INSERT on messages.typing_indicators.
-- Opportunistically removes all expired typing
-- indicator rows to keep the table lean. Expired
-- rows are those past their expires_at timestamp.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.cleanup_typing_indicator()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM messages.typing_indicators
    WHERE expires_at < NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;