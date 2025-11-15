-- =====================================================
-- MEDIA SCHEMA - FUNCTIONS
-- =====================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION media.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to increment file access count
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

-- Function to increment download count
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

-- Function to update album file count
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

-- Function to update sticker pack count
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

-- Function to update tag usage count
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

-- Function to update user storage stats
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
    -- Get totals
    SELECT 
        COUNT(*),
        COALESCE(SUM(file_size_bytes), 0)
    INTO v_total_files, v_total_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
    AND deleted_at IS NULL;
    
    -- Get images
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_images_count, v_images_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
    AND file_category = 'image'
    AND deleted_at IS NULL;
    
    -- Get videos
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_videos_count, v_videos_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
    AND file_category = 'video'
    AND deleted_at IS NULL;
    
    -- Get audio
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_audio_count, v_audio_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
    AND file_category = 'audio'
    AND deleted_at IS NULL;
    
    -- Get documents
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_documents_count, v_documents_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
    AND file_category = 'document'
    AND deleted_at IS NULL;
    
    -- Get quota
    SELECT storage_quota_bytes INTO v_quota
    FROM media.storage_stats
    WHERE user_id = p_user_id;
    
    -- Calculate percentage
    IF v_quota > 0 THEN
        v_percentage := (v_total_size::DECIMAL / v_quota * 100);
    ELSE
        v_percentage := 0;
    END IF;
    
    -- Update stats
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

-- Function to queue file processing
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

-- Function to clean up deleted files
CREATE OR REPLACE FUNCTION media.cleanup_deleted_files()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    -- Permanently delete files marked for deletion
    DELETE FROM media.files
    WHERE permanently_delete_at IS NOT NULL
    AND permanently_delete_at < NOW();
    
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to check storage quota
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
        v_quota := 5368709120; -- 5GB default
    END IF;
    
    RETURN (v_current_usage + p_file_size) <= v_quota;
END;
$$ LANGUAGE plpgsql;

-- Function to generate access token
CREATE OR REPLACE FUNCTION media.generate_access_token()
RETURNS TEXT AS $$
BEGIN
    RETURN encode(gen_random_bytes(32), 'base64');
END;
$$ LANGUAGE plpgsql;

-- Function to increment share counts
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

-- Function to increment sticker usage
CREATE OR REPLACE FUNCTION media.increment_sticker_usage(p_sticker_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE media.stickers
    SET usage_count = usage_count + 1
    WHERE id = p_sticker_id;
END;
$$ LANGUAGE plpgsql;

-- Function to increment GIF usage
CREATE OR REPLACE FUNCTION media.increment_gif_usage(p_gif_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE media.gifs
    SET usage_count = usage_count + 1
    WHERE id = p_gif_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get user files by type
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