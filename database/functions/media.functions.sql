-- =====================================================
-- MEDIA SCHEMA — UTILITY FUNCTIONS
-- =====================================================
--
-- Description:  Callable utility functions for the media
--               schema. These are invoked by application
--               code, trigger handler functions, or
--               periodic cleanup jobs — NOT directly by
--               triggers.
--
-- Note:         Trigger handler functions (RETURNS TRIGGER)
--               live in media.trigger_functions.sql.
--
-- Dependencies: media schema tables must exist.
--               pgcrypto extension (for gen_random_bytes).
--
-- =====================================================


-- -------------------------------------------------
-- media.update_storage_stats(p_user_id)
-- -------------------------------------------------
-- Fully recalculates the storage statistics for a
-- single user by scanning media.files. Updates (or
-- creates via UPSERT) the row in media.storage_stats
-- with per-category file counts, byte totals, and
-- storage_used_percentage.
--
-- Called by the trigger handler
-- media.trigger_update_storage_stats() and may also
-- be called directly from application code.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.update_storage_stats(p_user_id UUID)
RETURNS VOID AS $$
DECLARE
    v_total_files INTEGER;
    v_total_size BIGINT;
    v_images_count INTEGER;
    v_images_size BIGINT;
    v_videos_count INTEGER;
    v_videos_size BIGINT;
    v_audio_count INTEGER;
    v_audio_size BIGINT;
    v_documents_count INTEGER;
    v_documents_size BIGINT;
    v_quota BIGINT;
    v_percentage DECIMAL(5,2);
BEGIN
    -- Total across all categories
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_total_files, v_total_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND deleted_at IS NULL;

    -- Images
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_images_count, v_images_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND file_category = 'image'
      AND deleted_at IS NULL;

    -- Videos
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_videos_count, v_videos_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND file_category = 'video'
      AND deleted_at IS NULL;

    -- Audio
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_audio_count, v_audio_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND file_category = 'audio'
      AND deleted_at IS NULL;

    -- Documents
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_documents_count, v_documents_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND file_category = 'document'
      AND deleted_at IS NULL;

    -- Current quota
    SELECT storage_quota_bytes INTO v_quota
    FROM media.storage_stats
    WHERE user_id = p_user_id;

    -- Usage percentage
    IF v_quota > 0 THEN
        v_percentage := (v_total_size::DECIMAL / v_quota * 100);
    ELSE
        v_percentage := 0;
    END IF;

    -- Upsert storage stats
    INSERT INTO media.storage_stats (
        user_id, total_files, total_size_bytes,
        images_count, images_size_bytes,
        videos_count, videos_size_bytes,
        audio_count, audio_size_bytes,
        documents_count, documents_size_bytes,
        storage_used_percentage,
        last_calculated_at
    ) VALUES (
        p_user_id, v_total_files, v_total_size,
        v_images_count, v_images_size,
        v_videos_count, v_videos_size,
        v_audio_count, v_audio_size,
        v_documents_count, v_documents_size,
        v_percentage,
        NOW()
    )
    ON CONFLICT (user_id) DO UPDATE SET
        total_files = EXCLUDED.total_files,
        total_size_bytes = EXCLUDED.total_size_bytes,
        images_count = EXCLUDED.images_count,
        images_size_bytes = EXCLUDED.images_size_bytes,
        videos_count = EXCLUDED.videos_count,
        videos_size_bytes = EXCLUDED.videos_size_bytes,
        audio_count = EXCLUDED.audio_count,
        audio_size_bytes = EXCLUDED.audio_size_bytes,
        documents_count = EXCLUDED.documents_count,
        documents_size_bytes = EXCLUDED.documents_size_bytes,
        storage_used_percentage = EXCLUDED.storage_used_percentage,
        last_calculated_at = EXCLUDED.last_calculated_at,
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- media.queue_file_processing(...)
-- -------------------------------------------------
-- Inserts a row into media.processing_queue. Used by
-- trigger handlers (queue_thumbnail_generation,
-- queue_virus_scan) and may be called directly for
-- custom processing tasks.
--
-- Returns the UUID of the new processing_queue row.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.queue_file_processing(
    p_file_id UUID,
    p_task_type VARCHAR,
    p_priority INTEGER DEFAULT 5,
    p_input_params JSONB DEFAULT '{}'::JSONB
)
RETURNS UUID AS $$
DECLARE
    v_queue_id UUID;
BEGIN
    INSERT INTO media.processing_queue (
        file_id, task_type, priority, input_params
    ) VALUES (
        p_file_id, p_task_type, p_priority, p_input_params
    ) RETURNING id INTO v_queue_id;

    RETURN v_queue_id;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- media.cleanup_deleted_files()
-- -------------------------------------------------
-- Permanently deletes files whose
-- permanently_delete_at has passed. Called by a
-- periodic cleanup job (runs after the 30-day
-- soft-delete retention period).
--
-- Returns the number of files permanently deleted.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.cleanup_deleted_files()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM media.files
    WHERE permanently_delete_at IS NOT NULL
      AND permanently_delete_at < NOW();

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- media.check_storage_quota(p_user_id, p_file_size)
-- -------------------------------------------------
-- Returns TRUE if the user has enough remaining quota
-- to accommodate p_file_size additional bytes. Used
-- by the validate_storage_quota() trigger handler and
-- may be called from application code for pre-checks.
--
-- Default quota: 5 GB (5,368,709,120 bytes).
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.check_storage_quota(
    p_user_id UUID,
    p_file_size BIGINT
)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_usage BIGINT;
    v_quota BIGINT;
BEGIN
    SELECT total_size_bytes, storage_quota_bytes
    INTO v_current_usage, v_quota
    FROM media.storage_stats
    WHERE user_id = p_user_id;

    IF v_current_usage IS NULL THEN
        v_current_usage := 0;
        v_quota := 5368709120; -- 5 GB default
    END IF;

    RETURN (v_current_usage + p_file_size) <= v_quota;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- media.generate_access_token()
-- -------------------------------------------------
-- Generates a cryptographically random 32-byte token
-- encoded as base64. Used for share link access tokens.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.generate_access_token()
RETURNS TEXT AS $$
BEGIN
    RETURN encode(gen_random_bytes(32), 'base64');
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- media.increment_sticker_usage(p_sticker_id)
-- -------------------------------------------------
-- Increments the usage_count on a sticker. Called by
-- application code when a sticker is sent in a message.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.increment_sticker_usage(p_sticker_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE media.stickers
    SET usage_count = usage_count + 1
    WHERE id = p_sticker_id;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- media.increment_gif_usage(p_gif_id)
-- -------------------------------------------------
-- Increments the usage_count on a GIF. Called by
-- application code when a GIF is sent in a message.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.increment_gif_usage(p_gif_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE media.gifs
    SET usage_count = usage_count + 1
    WHERE id = p_gif_id;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- media.get_user_files(...)
-- -------------------------------------------------
-- Returns paginated file listing for a user,
-- optionally filtered by file_category. Only returns
-- non-deleted files, ordered newest first.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.get_user_files(
    p_user_id UUID,
    p_file_category VARCHAR DEFAULT NULL,
    p_limit INTEGER DEFAULT 50,
    p_offset INTEGER DEFAULT 0
)
RETURNS TABLE (
    file_id UUID,
    file_name VARCHAR,
    file_type VARCHAR,
    file_size BIGINT,
    storage_url TEXT,
    thumbnail_url TEXT,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        f.id,
        f.file_name,
        f.file_type,
        f.file_size_bytes,
        f.storage_url,
        f.thumbnail_url,
        f.created_at
    FROM media.files f
    WHERE f.uploader_user_id = p_user_id
      AND f.deleted_at IS NULL
      AND (p_file_category IS NULL OR f.file_category = p_file_category)
    ORDER BY f.created_at DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;
