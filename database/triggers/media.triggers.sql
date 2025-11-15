-- =====================================================
-- MEDIA SCHEMA - TRIGGERS
-- =====================================================

-- Trigger to update updated_at on media.files
CREATE TRIGGER trigger_media_files_updated_at
    BEFORE UPDATE ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Trigger to update updated_at on media.albums
CREATE TRIGGER trigger_media_albums_updated_at
    BEFORE UPDATE ON media.albums
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Trigger to update updated_at on media.sticker_packs
CREATE TRIGGER trigger_media_sticker_packs_updated_at
    BEFORE UPDATE ON media.sticker_packs
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Trigger to update updated_at on media.processing_queue
CREATE TRIGGER trigger_media_processing_queue_updated_at
    BEFORE UPDATE ON media.processing_queue
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Trigger to update updated_at on media.transcoding_jobs
CREATE TRIGGER trigger_media_transcoding_jobs_updated_at
    BEFORE UPDATE ON media.transcoding_jobs
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Trigger to update updated_at on media.storage_stats
CREATE TRIGGER trigger_media_storage_stats_updated_at
    BEFORE UPDATE ON media.storage_stats
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Trigger to increment access count
CREATE TRIGGER trigger_media_access_log_increment
    AFTER INSERT ON media.access_log
    FOR EACH ROW
    EXECUTE FUNCTION media.increment_access_count();

-- Trigger to increment download/view count
CREATE TRIGGER trigger_media_access_log_download
    AFTER INSERT ON media.access_log
    FOR EACH ROW
    WHEN (NEW.access_type IN ('download', 'view'))
    EXECUTE FUNCTION media.increment_download_count();

-- Trigger to update album file count
CREATE TRIGGER trigger_media_album_files_count_insert
    AFTER INSERT ON media.album_files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_album_file_count();

CREATE TRIGGER trigger_media_album_files_count_delete
    AFTER DELETE ON media.album_files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_album_file_count();

-- Trigger to update sticker pack count
CREATE TRIGGER trigger_media_stickers_count_insert
    AFTER INSERT ON media.stickers
    FOR EACH ROW
    EXECUTE FUNCTION media.update_sticker_pack_count();

CREATE TRIGGER trigger_media_stickers_count_update
    AFTER UPDATE ON media.stickers
    FOR EACH ROW
    WHEN (OLD.is_active IS DISTINCT FROM NEW.is_active)
    EXECUTE FUNCTION media.update_sticker_pack_count();

CREATE TRIGGER trigger_media_stickers_count_delete
    AFTER DELETE ON media.stickers
    FOR EACH ROW
    EXECUTE FUNCTION media.update_sticker_pack_count();

-- Trigger to update tag usage count
CREATE TRIGGER trigger_media_file_tags_usage_insert
    AFTER INSERT ON media.file_tags
    FOR EACH ROW
    EXECUTE FUNCTION media.update_tag_usage_count();

CREATE TRIGGER trigger_media_file_tags_usage_delete
    AFTER DELETE ON media.file_tags
    FOR EACH ROW
    EXECUTE FUNCTION media.update_tag_usage_count();

-- Trigger to update storage stats on file insert
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

CREATE TRIGGER trigger_media_files_storage_stats_insert
    AFTER INSERT ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.trigger_update_storage_stats();

CREATE TRIGGER trigger_media_files_storage_stats_update
    AFTER UPDATE ON media.files
    FOR EACH ROW
    WHEN (OLD.deleted_at IS DISTINCT FROM NEW.deleted_at OR OLD.file_size_bytes IS DISTINCT FROM NEW.file_size_bytes)
    EXECUTE FUNCTION media.trigger_update_storage_stats();

CREATE TRIGGER trigger_media_files_storage_stats_delete
    AFTER DELETE ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.trigger_update_storage_stats();

-- Trigger to queue thumbnail generation on file upload
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

CREATE TRIGGER trigger_media_files_queue_thumbnail
    AFTER INSERT ON media.files
    FOR EACH ROW
    WHEN (NEW.file_category IN ('image', 'video'))
    EXECUTE FUNCTION media.queue_thumbnail_generation();

-- Trigger to queue virus scan on file upload
CREATE OR REPLACE FUNCTION media.queue_virus_scan()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM media.queue_file_processing(
        NEW.id,
        'scan',
        10, -- High priority
        '{}'::JSONB
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_media_files_queue_scan
    AFTER INSERT ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.queue_virus_scan();

-- Trigger to set permanently_delete_at on soft delete
CREATE OR REPLACE FUNCTION media.set_permanent_deletion_date()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        NEW.permanently_delete_at = NEW.deleted_at + INTERVAL '30 days';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_media_files_set_deletion_date
    BEFORE UPDATE ON media.files
    FOR EACH ROW
    WHEN (NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL)
    EXECUTE FUNCTION media.set_permanent_deletion_date();

-- Trigger to increment share counts
CREATE TRIGGER trigger_media_access_log_share_counts
    AFTER INSERT ON media.access_log
    FOR EACH ROW
    WHEN (NEW.access_type IN ('view', 'download'))
    EXECUTE FUNCTION media.increment_share_counts();

-- Trigger to validate file size against quota
CREATE OR REPLACE FUNCTION media.validate_storage_quota()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT media.check_storage_quota(NEW.uploader_user_id, NEW.file_size_bytes) THEN
        RAISE EXCEPTION 'Storage quota exceeded for user %', NEW.uploader_user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_media_files_validate_quota
    BEFORE INSERT ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.validate_storage_quota();

-- Trigger to create default storage stats for new users
CREATE OR REPLACE FUNCTION media.create_default_storage_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO media.storage_stats (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_auth_users_create_storage_stats
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION media.create_default_storage_stats();