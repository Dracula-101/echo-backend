-- =====================================================
-- MESSAGES TRIGGERS — Delivery Tracking
-- =====================================================
--
-- Purpose:   Track message delivery lifecycle:
--              - Create delivery_status rows for all recipients
--              - Update delivery/read counts on messages
--              - Increment unread counts for recipients
--              - Increment mention counts for @-mentioned users
--
-- Depends:   functions/messages.trigger_functions.sql
--              -> messages.create_delivery_status()
--              -> messages.update_message_delivery_counts()
--              -> messages.increment_unread_count()
--              -> messages.increment_mention_count()
--
-- Tables:    messages.messages                (source for unread/mentions)
--            messages.delivery_status         (created & tracked)
--            messages.conversation_participants (counts updated)
--
-- =====================================================


-- -----------------------------------------------
-- DELIVERY STATUS CREATION
-- -----------------------------------------------

-- When a new message is sent, create a 'sent' delivery_status
-- row for every active participant except the sender.
-- This enables per-recipient delivery and read tracking.
CREATE TRIGGER trigger_messages_create_delivery_status
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.create_delivery_status();


-- -----------------------------------------------
-- DELIVERY COUNT UPDATES
-- -----------------------------------------------

-- When a delivery_status row is inserted or its status
-- changes, recalculate the delivery_count and read_count
-- on the parent message.
CREATE TRIGGER trigger_delivery_status_update_counts
    AFTER INSERT OR UPDATE ON messages.delivery_status
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_message_delivery_counts();


-- -----------------------------------------------
-- UNREAD & MENTION TRACKING
-- -----------------------------------------------

-- Increment the unread_count for all active participants
-- (except the sender) when a new message arrives.
CREATE TRIGGER trigger_messages_increment_unread
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.increment_unread_count();

-- When a message contains @-mentions (non-empty mentions JSONB),
-- increment the mention_count for each mentioned user's
-- participant record in the conversation.
CREATE TRIGGER trigger_messages_increment_mentions
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.mentions IS NOT NULL AND jsonb_array_length(NEW.mentions) > 0)
    EXECUTE FUNCTION messages.increment_mention_count();
