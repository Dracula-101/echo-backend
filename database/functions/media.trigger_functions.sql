-- =====================================================
-- MEDIA SCHEMA — TRIGGER HANDLER FUNCTIONS
-- =====================================================
--
-- Description:  Functions used exclusively as trigger handlers
--               for tables in the media schema.
--
-- Dependencies: media schema tables must exist.
--               media.update_storage_stats() utility function
--               (defined in media.functions.sql).
--               media.check_storage_quota() utility function
--               (defined in media.functions.sql).
--               media.queue_file_processing() utility function
--               (defined in media.functions.sql).
--
-- Execution:    Load AFTER media.functions.sql (utility functions)
--               and BEFORE any media trigger files.
--
-- =====================================================


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 1: TIMESTAMP MANAGEMENT
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- media.update_updated_at_column()
-- -------------------------------------------------
-- Automatically sets the `updated_at` column to the
-- current timestamp whenever a row is modified.
-- Used by: media.files, media.albums,
--          media.sticker_packs, media.processing_queue,
--          media.transcoding_jobs, media.storage_stats
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 2: STORAGE MANAGEMENT
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- media.trigger_update_storage_stats()
-- -------------------------------------------------
-- Fires AFTER INSERT, UPDATE, or DELETE on
-- media.files. Delegates to the
-- media.update_storage_stats() utility function to
-- recalculate the user's storage totals broken down
-- by file category (images, videos, audio, documents).
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.trigger_update_storage_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        PERFORM media.update_storage_stats(NEW.uploader_user_id);
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM media.update_storage_stats(OLD.uploader_user_id);
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- media.validate_storage_quota()
-- -------------------------------------------------
-- Fires BEFORE INSERT on media.files. Checks whether
-- the user has enough remaining storage quota for the
-- incoming file. Raises an exception if the quota
-- would be exceeded (default quota: 5 GB).
--
-- Depends on: media.check_storage_quota()
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.validate_storage_quota()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT media.check_storage_quota(NEW.uploader_user_id, NEW.file_size_bytes) THEN
        RAISE EXCEPTION 'Storage quota exceeded for user %', NEW.uploader_user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- media.create_default_storage_stats()
-- -------------------------------------------------
-- Fires AFTER INSERT on auth.users. Creates an
-- initial storage_stats row for every new user with
-- the default 5 GB quota and zero usage.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.create_default_storage_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO media.storage_stats (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- media.set_permanent_deletion_date()
-- -------------------------------------------------
-- Fires BEFORE UPDATE on media.files when deleted_at
-- transitions from NULL to non-NULL (soft delete).
-- Schedules permanent deletion 30 days in the future
-- by setting permanently_delete_at. A separate cleanup
-- job (media.cleanup_deleted_files) handles the actual
-- purge.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.set_permanent_deletion_date()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        NEW.permanently_delete_at = NEW.deleted_at + INTERVAL '30 days';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 3: PROCESSING PIPELINE
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- media.queue_thumbnail_generation()
-- -------------------------------------------------
-- Fires AFTER INSERT on media.files for image and
-- video files. Enqueues a 'thumbnail' processing task
-- at priority 5, requesting small, medium, and large
-- thumbnail sizes.
--
-- Depends on: media.queue_file_processing()
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.queue_thumbnail_generation()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.file_category IN ('image', 'video') THEN
        PERFORM media.queue_file_processing(
            NEW.id,
            'thumbnail',
            5,
            jsonb_build_object('sizes', ARRAY['small', 'medium', 'large'])
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- media.queue_virus_scan()
-- -------------------------------------------------
-- Fires AFTER INSERT on media.files for ALL files.
-- Enqueues a 'scan' processing task at priority 10
-- (highest) to ensure every uploaded file is scanned
-- for malware before it becomes accessible.
--
-- Depends on: media.queue_file_processing()
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.queue_virus_scan()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM media.queue_file_processing(
        NEW.id,
        'scan',
        10, -- High priority: scan before any other processing
        '{}'::JSONB
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- SECTION 4: COUNT TRACKING
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- -------------------------------------------------
-- media.increment_access_count()
-- -------------------------------------------------
-- Fires AFTER INSERT on media.access_log. Increments
-- the access_count on the file and updates
-- last_accessed_at to track file popularity.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.increment_access_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE media.files
    SET access_count = access_count + 1,
        last_accessed_at = NOW()
    WHERE id = NEW.file_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- media.increment_download_count()
-- -------------------------------------------------
-- Fires AFTER INSERT on media.access_log when the
-- access_type is 'download' or 'view'. Increments
-- the corresponding counter on media.files.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.increment_download_count()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.access_type = 'download' THEN
        UPDATE media.files
        SET download_count = download_count + 1
        WHERE id = NEW.file_id;
    ELSIF NEW.access_type = 'view' THEN
        UPDATE media.files
        SET view_count = view_count + 1
        WHERE id = NEW.file_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- media.update_album_file_count()
-- -------------------------------------------------
-- Fires AFTER INSERT or DELETE on media.album_files.
-- Recalculates the file_count on the parent album.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.update_album_file_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE media.albums
    SET file_count = (
        SELECT COUNT(*) FROM media.album_files
        WHERE album_id = COALESCE(NEW.album_id, OLD.album_id)
    ),
    updated_at = NOW()
    WHERE id = COALESCE(NEW.album_id, OLD.album_id);

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- media.update_sticker_pack_count()
-- -------------------------------------------------
-- Fires AFTER INSERT, UPDATE (is_active change), or
-- DELETE on media.stickers. Recalculates the
-- sticker_count on the parent sticker pack, counting
-- only active stickers.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.update_sticker_pack_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE media.sticker_packs
    SET sticker_count = (
        SELECT COUNT(*) FROM media.stickers
        WHERE sticker_pack_id = COALESCE(NEW.sticker_pack_id, OLD.sticker_pack_id)
          AND is_active = TRUE
    ),
    updated_at = NOW()
    WHERE id = COALESCE(NEW.sticker_pack_id, OLD.sticker_pack_id);

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- media.update_tag_usage_count()
-- -------------------------------------------------
-- Fires AFTER INSERT or DELETE on media.file_tags.
-- Recalculates the usage_count on the affected tag
-- to reflect how many files use it.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.update_tag_usage_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE media.tags
    SET usage_count = (
        SELECT COUNT(*) FROM media.file_tags
        WHERE tag_id = COALESCE(NEW.tag_id, OLD.tag_id)
    )
    WHERE id = COALESCE(NEW.tag_id, OLD.tag_id);

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------
-- media.increment_share_counts()
-- -------------------------------------------------
-- Fires AFTER INSERT on media.access_log when the
-- access_type is 'view' or 'download'. Increments
-- the corresponding counter on any active share
-- records for the accessed file.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION media.increment_share_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.access_type = 'view' THEN
            UPDATE media.shares
            SET view_count = view_count + 1
            WHERE file_id = NEW.file_id
              AND is_active = TRUE;
        ELSIF NEW.access_type = 'download' THEN
            UPDATE media.shares
            SET download_count = download_count + 1
            WHERE file_id = NEW.file_id
              AND is_active = TRUE;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
