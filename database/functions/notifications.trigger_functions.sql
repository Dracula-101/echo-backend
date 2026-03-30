-- =====================================================
-- NOTIFICATIONS SCHEMA — TRIGGER HANDLER FUNCTIONS
-- =====================================================
--
-- Description:  Functions used exclusively as trigger handlers
--               for tables in the notifications schema.
--
-- Dependencies: notifications schema tables must exist.
--               messages.conversation_participants table
--               must exist (for channel validation).
--
-- Execution:    Load AFTER notifications.functions.sql
--               (utility functions) and BEFORE any
--               notifications trigger files.
--
-- =====================================================


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 1: TIMESTAMP MANAGEMENT
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- notifications.update_updated_at_column()
-- -------------------------------------------------
-- Automatically sets the `updated_at` column to the
-- current timestamp whenever a row is modified.
-- Used by: notifications.notifications,
--          notifications.user_preferences,
--          notifications.conversation_channels,
--          notifications.templates,
--          notifications.announcements,
--          notifications.user_stats
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 2: USER REGISTRATION SETUP
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- notifications.create_default_preferences()
-- -------------------------------------------------
-- Fires AFTER INSERT on auth.users. Creates a default
-- notification preferences row for the new user with
-- all preference columns at their DEFAULT values.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.create_default_preferences()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO notifications.user_preferences (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- notifications.create_default_notification_stats()
-- -------------------------------------------------
-- Fires AFTER INSERT on auth.users. Creates a default
-- notification statistics row for the new user with
-- all counters initialized to zero.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.create_default_notification_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO notifications.user_stats (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 3: STATISTICS TRACKING
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- notifications.update_user_stats()
-- -------------------------------------------------
-- Fires AFTER INSERT on notifications.notifications.
-- Increments the total_notifications_sent counter and
-- conditionally increments push_sent if the notification
-- targets a mobile platform (ios or android).
-- Uses UPSERT to handle the first notification for a user.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.update_user_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO notifications.user_stats (
            user_id,
            total_notifications_sent,
            push_sent,
            last_notification_at
        ) VALUES (
            NEW.user_id,
            1,
            CASE WHEN NEW.platform IN ('ios', 'android') THEN 1 ELSE 0 END,
            NEW.created_at
        )
        ON CONFLICT (user_id) DO UPDATE SET
            total_notifications_sent = notifications.user_stats.total_notifications_sent + 1,
            push_sent = notifications.user_stats.push_sent +
                CASE WHEN NEW.platform IN ('ios', 'android') THEN 1 ELSE 0 END,
            last_notification_at = NEW.created_at,
            updated_at = NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- notifications.update_delivery_stats()
-- -------------------------------------------------
-- Fires AFTER UPDATE on notifications.push_delivery_log
-- when the status or opened_at columns change.
-- Increments the appropriate delivery/opened counters
-- on notifications.user_stats.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.update_delivery_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'delivered' AND (OLD.status IS NULL OR OLD.status != 'delivered') THEN
        UPDATE notifications.user_stats
        SET total_notifications_delivered = total_notifications_delivered + 1,
            push_delivered = push_delivered + 1,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    ELSIF NEW.status = 'opened' AND (OLD.opened_at IS NULL) THEN
        UPDATE notifications.user_stats
        SET total_notifications_opened = total_notifications_opened + 1,
            push_opened = push_opened + 1,
            last_opened_notification_at = NEW.opened_at,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- notifications.update_email_stats()
-- -------------------------------------------------
-- Fires AFTER UPDATE on notifications.email_notifications
-- when status, opened_at, or clicked_at changes.
-- Increments the appropriate email delivery/opened/clicked
-- counters on notifications.user_stats.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.update_email_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'delivered' AND (OLD.status IS NULL OR OLD.status != 'delivered') THEN
        UPDATE notifications.user_stats
        SET email_delivered = email_delivered + 1,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    ELSIF NEW.opened_at IS NOT NULL AND OLD.opened_at IS NULL THEN
        UPDATE notifications.user_stats
        SET email_opened = email_opened + 1,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    ELSIF NEW.clicked_at IS NOT NULL AND OLD.clicked_at IS NULL THEN
        UPDATE notifications.user_stats
        SET email_clicked = email_clicked + 1,
            updated_at = NOW()
        WHERE user_id = NEW.user_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 4: DELIVERY LIFECYCLE
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- notifications.update_notification_delivery_status()
-- -------------------------------------------------
-- Fires AFTER UPDATE on notifications.push_delivery_log
-- when status changes. Syncs the delivery status back
-- to the parent notifications.notifications row,
-- including the delivered_at timestamp or failure reason.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.update_notification_delivery_status()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'delivered' AND (OLD.status IS NULL OR OLD.status != 'delivered') THEN
        UPDATE notifications.notifications
        SET delivery_status = 'delivered',
            delivered_at = NEW.delivered_at
        WHERE id = NEW.notification_id;
    ELSIF NEW.status = 'failed' AND (OLD.status IS NULL OR OLD.status != 'failed') THEN
        UPDATE notifications.notifications
        SET delivery_status = 'failed',
            failed_reason = NEW.error_message
        WHERE id = NEW.notification_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- notifications.update_batch_progress()
-- -------------------------------------------------
-- Fires AFTER UPDATE on notifications.notifications
-- when delivery_status or is_read changes. Updates
-- the sent/delivered/failed/opened counters on the
-- parent batch record (if the notification belongs
-- to a batch).
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.update_batch_progress()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.delivery_status = 'sent' AND (OLD.delivery_status IS NULL OR OLD.delivery_status != 'sent') THEN
        UPDATE notifications.batches
        SET sent_count = sent_count + 1
        WHERE id IN (
            SELECT jsonb_array_elements_text(metadata->'batch_id')::UUID
            FROM notifications.notifications
            WHERE id = NEW.id
        );
    ELSIF NEW.delivery_status = 'delivered' AND (OLD.delivery_status IS NULL OR OLD.delivery_status != 'delivered') THEN
        UPDATE notifications.batches
        SET delivered_count = delivered_count + 1
        WHERE id IN (
            SELECT jsonb_array_elements_text(metadata->'batch_id')::UUID
            FROM notifications.notifications
            WHERE id = NEW.id
        );
    ELSIF NEW.delivery_status = 'failed' AND (OLD.delivery_status IS NULL OR OLD.delivery_status != 'failed') THEN
        UPDATE notifications.batches
        SET failed_count = failed_count + 1
        WHERE id IN (
            SELECT jsonb_array_elements_text(metadata->'batch_id')::UUID
            FROM notifications.notifications
            WHERE id = NEW.id
        );
    END IF;

    IF NEW.is_read = TRUE AND OLD.is_read = FALSE THEN
        UPDATE notifications.batches
        SET opened_count = opened_count + 1
        WHERE id IN (
            SELECT jsonb_array_elements_text(metadata->'batch_id')::UUID
            FROM notifications.notifications
            WHERE id = NEW.id
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 5: ANNOUNCEMENTS
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- notifications.increment_announcement_views()
-- -------------------------------------------------
-- Fires AFTER INSERT on notifications.announcement_views.
-- Increments the view_count on the parent announcement.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.increment_announcement_views()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE notifications.announcements
    SET view_count = view_count + 1
    WHERE id = NEW.announcement_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- notifications.increment_announcement_clicks()
-- -------------------------------------------------
-- Fires AFTER UPDATE on notifications.announcement_views
-- when clicked or dismissed transitions to TRUE.
-- Increments click_count and/or dismiss_count on the
-- parent announcement.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.increment_announcement_clicks()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.clicked = TRUE AND OLD.clicked = FALSE THEN
        UPDATE notifications.announcements
        SET click_count = click_count + 1
        WHERE id = NEW.announcement_id;
    END IF;

    IF NEW.dismissed = TRUE AND OLD.dismissed = FALSE THEN
        UPDATE notifications.announcements
        SET dismiss_count = dismiss_count + 1
        WHERE id = NEW.announcement_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 6: VALIDATION
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- notifications.validate_conversation_channel()
-- -------------------------------------------------
-- Fires BEFORE INSERT or UPDATE on
-- notifications.conversation_channels. Ensures the
-- user is an active participant in the referenced
-- conversation before allowing channel notification
-- overrides.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION notifications.validate_conversation_channel()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = NEW.conversation_id
          AND user_id = NEW.user_id
          AND left_at IS NULL
          AND removed_at IS NULL
    ) THEN
        RAISE EXCEPTION 'User % is not a participant of conversation %',
            NEW.user_id, NEW.conversation_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
