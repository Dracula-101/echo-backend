-- database-dup/functions/auth.functions.sql
CREATE OR REPLACE FUNCTION auth.generate_device_fingerprint(
    p_device_id TEXT,
    p_device_name TEXT,
    p_device_type TEXT,
    p_device_os TEXT,
    p_device_os_version TEXT,
    p_device_model TEXT,
    p_device_manufacturer TEXT
)
RETURNS TEXT
AS $$
DECLARE
    v_fingerprint_source TEXT;
    v_fingerprint TEXT;
BEGIN
    v_fingerprint_source := COALESCE(p_device_id, '') || '|' ||
                            COALESCE(p_device_name, '') || '|' ||
                            COALESCE(p_device_type, '') || '|' ||
                            COALESCE(p_device_os, '') || '|' ||
                            COALESCE(p_device_os_version, '') || '|' ||
                            COALESCE(p_device_model, '') || '|' ||
                            COALESCE(p_device_manufacturer, '');
    v_fingerprint := encode(digest(v_fingerprint_source, 'sha256'), 'hex');
    RETURN v_fingerprint;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.log_security_event(
    p_user_id UUID,
    p_session_id UUID,
    p_event_type VARCHAR,
    p_event_category VARCHAR,
    p_severity VARCHAR,
    p_status VARCHAR,
    p_description TEXT,
    p_ip_address INET,
    p_user_agent TEXT,
    p_device_id TEXT,
    p_location_country VARCHAR,
    p_location_city VARCHAR,
    p_metadata JSONB
)
RETURNS UUID AS $$
DECLARE
    v_event_id UUID;
BEGIN
    INSERT INTO auth.security_events (
        user_id, session_id, event_type, event_category,
        severity, status, description, ip_address,
        user_agent, device_id, location_country, location_city, metadata
    ) VALUES (
        p_user_id, p_session_id, p_event_type, p_event_category,
        p_severity, p_status, p_description, p_ip_address,
        p_user_agent, p_device_id, p_location_country, p_location_city, p_metadata
    ) RETURNING id INTO v_event_id;
    RETURN v_event_id;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.soft_delete_user(p_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_user RECORD;
    v_revoked_sessions INTEGER;
BEGIN
    SELECT id, account_status, deleted_at
    INTO v_user
    FROM auth.users
    WHERE id = p_user_id;
    IF v_user IS NULL THEN
        RAISE EXCEPTION 'User % does not exist', p_user_id;
    END IF;
    IF v_user.account_status = 'deleted' OR v_user.deleted_at IS NOT NULL THEN
        RAISE EXCEPTION 'User % is already deleted', p_user_id;
    END IF;
    UPDATE auth.sessions
    SET revoked_at = NOW(),
        revoked_reason = 'account_deleted'
    WHERE user_id = p_user_id
      AND revoked_at IS NULL;
    GET DIAGNOSTICS v_revoked_sessions = ROW_COUNT;
    UPDATE auth.users
    SET account_status = 'deleted',
        deleted_at = NOW(),
        updated_at = NOW()
    WHERE id = p_user_id;
    UPDATE users.profiles
    SET deactivated_at = NOW(),
        online_status = 'offline',
        display_name = 'Deleted User',
        bio = NULL,
        avatar_url = NULL,
        cover_image_url = NULL,
        website_url = NULL,
        location = NULL,
        search_visibility = FALSE,
        updated_at = NOW()
    WHERE user_id = p_user_id;
    UPDATE media.files
    SET deleted_at = NOW(),
        permanently_delete_at = NOW() + INTERVAL '30 days'
    WHERE uploader_user_id = p_user_id
      AND deleted_at IS NULL;
    PERFORM auth.log_security_event(
        p_user_id,
        NULL,                          
        'account_soft_delete',
        'account_management',
        'warning',
        'success',
        'User account soft-deleted. Sessions revoked: ' || v_revoked_sessions,
        NULL, NULL, NULL, NULL, NULL,
        jsonb_build_object(
            'deletion_type', 'soft',
            'sessions_revoked', v_revoked_sessions
        )
    );
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.hard_delete_user(p_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_user_exists BOOLEAN;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM auth.users WHERE id = p_user_id
    ) INTO v_user_exists;
    IF NOT v_user_exists THEN
        RAISE EXCEPTION 'User % does not exist', p_user_id;
    END IF;
    UPDATE auth.sessions
    SET revoked_at = NOW(),
        revoked_reason = 'account_hard_deleted'
    WHERE user_id = p_user_id
      AND revoked_at IS NULL;
    PERFORM auth.log_security_event(
        p_user_id,
        NULL,
        'account_hard_delete',
        'account_management',
        'critical',
        'success',
        'User account permanently deleted (GDPR purge)',
        NULL, NULL, NULL, NULL, NULL,
        jsonb_build_object(
            'deletion_type', 'hard',
            'gdpr_purge', TRUE
        )
    );
    DELETE FROM messages.reactions
    WHERE user_id = p_user_id;
    DELETE FROM messages.delivery_status
    WHERE user_id = p_user_id;
    DELETE FROM messages.poll_votes
    WHERE user_id = p_user_id;
    DELETE FROM messages.call_participants
    WHERE user_id = p_user_id;
    DELETE FROM messages.pinned_messages
    WHERE pinned_by_user_id = p_user_id;
    DELETE FROM messages.message_reports
    WHERE reporter_user_id = p_user_id;
    UPDATE messages.calls
    SET initiator_user_id = NULL
    WHERE initiator_user_id = p_user_id;
    DELETE FROM notifications.action_responses
    WHERE user_id = p_user_id;
    DELETE FROM notifications.announcement_views
    WHERE user_id = p_user_id;
    UPDATE notifications.announcements
    SET created_by_user_id = NULL
    WHERE created_by_user_id = p_user_id;
    UPDATE notifications.batches
    SET created_by_user_id = NULL
    WHERE created_by_user_id = p_user_id;
    UPDATE notifications.templates
    SET created_by_user_id = NULL
    WHERE created_by_user_id = p_user_id;
    DELETE FROM users.reports
    WHERE reported_user_id = p_user_id;
    UPDATE users.reports
    SET assigned_to = NULL
    WHERE assigned_to = p_user_id;
    UPDATE analytics.ab_tests
    SET created_by_user_id = NULL
    WHERE created_by_user_id = p_user_id;
    DELETE FROM auth.users
    WHERE id = p_user_id;
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- database-dup/functions/auth.trigger_functions.sql
CREATE OR REPLACE FUNCTION auth.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.update_failed_login_attempts()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'failed' THEN
        UPDATE auth.users
        SET failed_login_attempts = failed_login_attempts + 1,
            last_failed_login_at = NOW()
        WHERE id = NEW.user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.update_last_successful_login()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'success' THEN
        UPDATE auth.users
        SET last_successful_login_at = NOW(),
            failed_login_attempts = 0
        WHERE id = NEW.user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.update_device_fingerprint()
RETURNS TRIGGER AS $$
BEGIN
    DECLARE
        v_device_info RECORD;
    BEGIN
        SELECT device_id, device_name, device_type,
               device_os, device_os_version,
               device_model, device_manufacturer
        INTO v_device_info
        FROM auth.sessions
        WHERE id = NEW.session_id;
        NEW.device_fingerprint = auth.generate_device_fingerprint(
            v_device_info.device_id,
            v_device_info.device_name,
            v_device_info.device_type,
            v_device_info.device_os,
            v_device_info.device_os_version,
            v_device_info.device_model,
            v_device_info.device_manufacturer
        );
    END;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.log_password_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.password_hash IS DISTINCT FROM NEW.password_hash THEN
        PERFORM auth.log_security_event(
            NEW.id,
            NULL,
            'password_change',
            'account_management',
            'info',
            'success',
            'User password was changed',
            NULL,
            NULL,
            NULL,
            NULL,
            NULL,
            '{}'::JSONB
        );
        NEW.password_last_changed_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.log_2fa_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.two_factor_enabled IS DISTINCT FROM NEW.two_factor_enabled THEN
        PERFORM auth.log_security_event(
            NEW.id,
            NULL,
            CASE WHEN NEW.two_factor_enabled THEN '2fa_enable' ELSE '2fa_disable' END,
            'security',
            'info',
            'success',
            CASE WHEN NEW.two_factor_enabled
                 THEN 'Two-factor authentication enabled'
                 ELSE 'Two-factor authentication disabled'
            END,
            NULL,
            NULL,
            NULL,
            NULL,
            NULL,
            '{}'::JSONB
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.log_failed_login_attempt()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'failure' THEN
        UPDATE auth.users
        SET failed_login_attempts = failed_login_attempts + 1,
            last_failed_login_at = NOW()
        WHERE id = NEW.user_id;
        PERFORM auth.log_security_event(
            NEW.user_id,
            NULL,
            'failed_login',
            'authentication',
            'warning',
            'failure',
            'Failed login attempt',
            NEW.ip_address,
            NEW.user_agent,
            NEW.device_id,
            NEW.location_country,
            NEW.location_city,
            jsonb_build_object(
                'reason', NEW.failure_reason
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.log_session_creation()
RETURNS TRIGGER AS $$
DECLARE
    v_login_history_row RECORD;
BEGIN
    UPDATE auth.users
    SET last_successful_login_at = NOW(),
        failed_login_attempts = 0
    WHERE id = NEW.user_id;
    INSERT INTO auth.login_history (
        user_id, session_id, login_method, status,
        ip_address, user_agent, device_id,
        location_country, location_city, latitude, longitude,
        is_new_device, is_new_location
    ) VALUES (
        NEW.user_id, NEW.id, 'password', 'success',
        NEW.ip_address, NEW.user_agent, NEW.device_id,
        NEW.ip_country, NEW.ip_city, NEW.latitude, NEW.longitude,
        NOT NEW.is_trusted_device, FALSE
    ) RETURNING * INTO v_login_history_row;
    PERFORM auth.log_security_event(
        NEW.user_id,
        NEW.id,
        'login',
        'authentication',
        'info',
        'success',
        'User logged in successfully',
        NEW.ip_address,
        NEW.user_agent,
        NEW.device_id,
        NEW.ip_country,
        NEW.ip_city,
        jsonb_build_object(
            'is_new_device', NOT NEW.is_trusted_device,
            'device_type', NEW.device_type,
            'device_fingerprint', v_login_history_row.device_fingerprint
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.log_session_revocation()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.revoked_at IS NOT NULL AND OLD.revoked_at IS NULL THEN
        PERFORM auth.log_security_event(
            NEW.user_id,
            NEW.id,
            'logout',
            'authentication',
            'info',
            'success',
            'Session revoked: ' || COALESCE(NEW.revoked_reason, 'User logged out'),
            NEW.ip_address,
            NEW.user_agent,
            NEW.device_id,
            NEW.ip_country,
            NEW.ip_city,
            jsonb_build_object()
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.prevent_user_deletion_with_active_sessions()
RETURNS TRIGGER AS $$
DECLARE
    v_active_sessions INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_active_sessions
    FROM auth.sessions
    WHERE user_id = OLD.id
      AND revoked_at IS NULL
      AND expires_at > NOW();
    IF v_active_sessions > 0 THEN
        RAISE EXCEPTION 'Cannot delete user with active sessions. Revoke all sessions first.';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION auth.cleanup_user_oauth_providers()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM auth.oauth_providers WHERE user_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- database-dup/functions/media.functions.sql
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
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_total_files, v_total_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND deleted_at IS NULL;
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_images_count, v_images_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND file_category = 'image'
      AND deleted_at IS NULL;
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_videos_count, v_videos_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND file_category = 'video'
      AND deleted_at IS NULL;
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_audio_count, v_audio_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND file_category = 'audio'
      AND deleted_at IS NULL;
    SELECT COUNT(*), COALESCE(SUM(file_size_bytes), 0)
    INTO v_documents_count, v_documents_size
    FROM media.files
    WHERE uploader_user_id = p_user_id
      AND file_category = 'document'
      AND deleted_at IS NULL;
    SELECT storage_quota_bytes INTO v_quota
    FROM media.storage_stats
    WHERE user_id = p_user_id;
    IF v_quota > 0 THEN
        v_percentage := (v_total_size::DECIMAL / v_quota * 100);
    ELSE
        v_percentage := 0;
    END IF;
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
        v_quota := 5368709120; 
    END IF;
    RETURN (v_current_usage + p_file_size) <= v_quota;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION media.generate_access_token()
RETURNS TEXT AS $$
BEGIN
    RETURN encode(gen_random_bytes(32), 'base64');
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION media.increment_sticker_usage(p_sticker_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE media.stickers
    SET usage_count = usage_count + 1
    WHERE id = p_sticker_id;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION media.increment_gif_usage(p_gif_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE media.gifs
    SET usage_count = usage_count + 1
    WHERE id = p_gif_id;
END;
$$ LANGUAGE plpgsql;
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

-- database-dup/functions/media.trigger_functions.sql
CREATE OR REPLACE FUNCTION media.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
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
CREATE OR REPLACE FUNCTION media.validate_storage_quota()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT media.check_storage_quota(NEW.uploader_user_id, NEW.file_size_bytes) THEN
        RAISE EXCEPTION 'Storage quota exceeded for user %', NEW.uploader_user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION media.create_default_storage_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO media.storage_stats (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION media.set_permanent_deletion_date()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        NEW.permanently_delete_at = NEW.deleted_at + INTERVAL '30 days';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
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
CREATE OR REPLACE FUNCTION media.queue_virus_scan()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM media.queue_file_processing(
        NEW.id,
        'scan',
        10, 
        '{}'::JSONB
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
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

-- database-dup/functions/messages.functions.sql
CREATE OR REPLACE FUNCTION messages.mark_messages_as_read(
    p_user_id         UUID,
    p_conversation_id UUID,
    p_message_id      UUID
)
RETURNS VOID AS $$
BEGIN
    UPDATE messages.delivery_status ds
    SET status  = 'read',
        read_at = NOW()
    WHERE ds.user_id    = p_user_id
      AND ds.status    != 'read'
      AND ds.message_id IN (
          SELECT m.id
          FROM messages.messages m
          WHERE m.conversation_id = p_conversation_id
            AND m.id             <= p_message_id
      );
    UPDATE messages.conversation_participants
    SET last_read_message_id = p_message_id,
        last_read_at         = NOW(),
        unread_count         = 0,
        mention_count        = 0
    WHERE user_id         = p_user_id
      AND conversation_id = p_conversation_id;
    UPDATE messages.messages m
    SET read_count = (
        SELECT COUNT(*)
        FROM messages.delivery_status ds
        WHERE ds.message_id = m.id
          AND ds.status     = 'read'
    )
    WHERE m.conversation_id = p_conversation_id
      AND m.id             <= p_message_id;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION messages.delete_message(
    p_message_id UUID,
    p_user_id    UUID,
    p_delete_for VARCHAR DEFAULT 'sender'
)
RETURNS VOID AS $$
BEGIN
    IF p_delete_for = 'everyone' THEN
        UPDATE messages.messages
        SET is_deleted   = TRUE,
            deleted_at   = NOW(),
            deleted_for  = 'everyone',
            content      = '[Message deleted]'
        WHERE id             = p_message_id
          AND sender_user_id = p_user_id
          AND is_deleted     = FALSE;
    ELSE
        UPDATE messages.messages
        SET is_deleted   = TRUE,
            deleted_at   = NOW(),
            deleted_for  = 'sender'
        WHERE id             = p_message_id
          AND sender_user_id = p_user_id
          AND is_deleted     = FALSE;
    END IF;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION messages.get_conversation_messages(
    p_conversation_id    UUID,
    p_user_id            UUID,
    p_limit              INTEGER DEFAULT 50,
    p_before_message_id  UUID    DEFAULT NULL
)
RETURNS TABLE (
    message_id       UUID,
    sender_id        UUID,
    sender_username  VARCHAR,
    sender_avatar    TEXT,
    content          TEXT,
    message_type     VARCHAR,
    created_at       TIMESTAMPTZ,
    is_edited        BOOLEAN,
    reactions        JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.id,
        m.sender_user_id,
        p.username,
        p.avatar_url,
        CASE
            WHEN m.is_deleted THEN '[Message deleted]'
            ELSE m.content
        END,
        m.message_type,
        m.created_at,
        m.is_edited,
        COALESCE(
            (
                SELECT jsonb_agg(
                    jsonb_build_object(
                        'user_id',       r.user_id,
                        'reaction_type', r.reaction_type,
                        'created_at',    r.created_at
                    )
                    ORDER BY r.created_at
                )
                FROM messages.reactions r
                WHERE r.message_id = m.id
            ),
            '[]'::JSONB
        )
    FROM messages.messages m
    JOIN users.profiles p ON p.user_id = m.sender_user_id
    WHERE m.conversation_id = p_conversation_id
      AND (p_before_message_id IS NULL OR m.id < p_before_message_id)
      AND NOT EXISTS (
          SELECT 1
          FROM users.blocked_users b
          WHERE (b.user_id         = p_user_id          AND b.blocked_user_id = m.sender_user_id)
             OR (b.user_id         = m.sender_user_id   AND b.blocked_user_id = p_user_id)
      )
    ORDER BY m.created_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION messages.cleanup_expired_typing_indicators()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM messages.typing_indicators
    WHERE expires_at < NOW();
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION messages.expire_disappearing_messages()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    UPDATE messages.messages
    SET is_deleted   = TRUE,
        deleted_at   = NOW(),
        deleted_for  = 'everyone',
        content      = '[Message expired]'
    WHERE expires_at < NOW()
      AND is_deleted  = FALSE;
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;
-- database-dup/functions/messages.trigger_functions.sql
CREATE OR REPLACE FUNCTION messages.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION messages.create_default_conversation_settings()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.conversation_settings (conversation_id)
    VALUES (NEW.id)
    ON CONFLICT (conversation_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
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
CREATE OR REPLACE FUNCTION messages.update_search_index()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_deleted = TRUE THEN
        DELETE FROM messages.search_index WHERE message_id = NEW.id;
        RETURN NEW;
    END IF;
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
CREATE OR REPLACE FUNCTION messages.update_poll_votes()
RETURNS TRIGGER AS $$
DECLARE
    v_poll_option_id UUID;
    v_poll_id        UUID;
BEGIN
    v_poll_option_id := COALESCE(NEW.poll_option_id, OLD.poll_option_id);
    v_poll_id        := COALESCE(NEW.poll_id,        OLD.poll_id);
    UPDATE messages.poll_options
    SET vote_count = (
        SELECT COUNT(*)
        FROM messages.poll_votes
        WHERE poll_option_id = v_poll_option_id
    )
    WHERE id = v_poll_option_id;
    UPDATE messages.polls
    SET total_votes = (
        SELECT COUNT(*)
        FROM messages.poll_votes
        WHERE poll_id = v_poll_id
    )
    WHERE id = v_poll_id;
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
CREATE OR REPLACE FUNCTION messages.validate_not_blocked()
RETURNS TRIGGER AS $$
DECLARE
    v_recipient_id UUID;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM messages.conversations
        WHERE id                = NEW.conversation_id
          AND conversation_type = 'direct'
    ) THEN
        RETURN NEW;
    END IF;
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
CREATE OR REPLACE FUNCTION messages.cleanup_typing_indicator()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM messages.typing_indicators
    WHERE expires_at < NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- database-dup/functions/notifications.functions.sql
CREATE OR REPLACE FUNCTION notifications.mark_as_read(p_notification_id UUID, p_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE notifications.notifications
    SET is_read = TRUE,
        read_at = NOW()
    WHERE id = p_notification_id
      AND user_id = p_user_id
      AND is_read = FALSE;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION notifications.mark_as_seen(p_notification_id UUID, p_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE notifications.notifications
    SET is_seen = TRUE,
        seen_at = NOW()
    WHERE id = p_notification_id
      AND user_id = p_user_id
      AND is_seen = FALSE;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION notifications.mark_all_as_read(p_user_id UUID)
RETURNS INTEGER AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    UPDATE notifications.notifications
    SET is_read = TRUE,
        read_at = NOW()
    WHERE user_id = p_user_id
      AND is_read = FALSE
      AND deleted_at IS NULL;
    GET DIAGNOSTICS v_updated_count = ROW_COUNT;
    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION notifications.get_unread_count(p_user_id UUID)
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM notifications.notifications
    WHERE user_id = p_user_id
      AND is_read = FALSE
      AND deleted_at IS NULL
      AND (expires_at IS NULL OR expires_at > NOW());
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION notifications.cleanup_expired_notifications()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    UPDATE notifications.notifications
    SET deleted_at = NOW()
    WHERE expires_at < NOW()
      AND deleted_at IS NULL;
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION notifications.can_receive_notification(
    p_user_id UUID,
    p_notification_type VARCHAR,
    p_channel VARCHAR -- 'push', 'email', 'sms'
)
RETURNS BOOLEAN AS $$
DECLARE
    v_prefs RECORD;
    v_can_receive BOOLEAN := FALSE;
BEGIN
    SELECT * INTO v_prefs
    FROM notifications.user_preferences
    WHERE user_id = p_user_id;
    IF v_prefs IS NULL THEN
        RETURN TRUE;
    END IF;
    IF v_prefs.quiet_hours_enabled THEN
        IF CURRENT_TIME BETWEEN v_prefs.quiet_hours_start AND v_prefs.quiet_hours_end THEN
            RETURN FALSE;
        END IF;
    END IF;
    IF p_channel = 'push' AND NOT v_prefs.push_enabled THEN
        RETURN FALSE;
    ELSIF p_channel = 'email' AND NOT v_prefs.email_enabled THEN
        RETURN FALSE;
    ELSIF p_channel = 'sms' AND NOT v_prefs.sms_enabled THEN
        RETURN FALSE;
    END IF;
    IF p_notification_type = 'message' THEN
        v_can_receive := CASE p_channel
            WHEN 'push' THEN v_prefs.message_push
            WHEN 'email' THEN v_prefs.message_email
            WHEN 'sms' THEN v_prefs.message_sms
            ELSE FALSE
        END;
    ELSIF p_notification_type = 'mention' THEN
        v_can_receive := CASE p_channel
            WHEN 'push' THEN v_prefs.mention_push
            WHEN 'email' THEN v_prefs.mention_email
            WHEN 'sms' THEN v_prefs.mention_sms
            ELSE FALSE
        END;
    ELSIF p_notification_type = 'call' THEN
        v_can_receive := CASE p_channel
            WHEN 'push' THEN v_prefs.call_push
            WHEN 'email' THEN v_prefs.call_email
            WHEN 'sms' THEN v_prefs.call_sms
            ELSE FALSE
        END;
    ELSE
        v_can_receive := TRUE; 
    END IF;
    RETURN v_can_receive;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION notifications.create_notification(
    p_user_id UUID,
    p_notification_type VARCHAR,
    p_title VARCHAR,
    p_body TEXT,
    p_related_user_id UUID DEFAULT NULL,
    p_related_message_id UUID DEFAULT NULL,
    p_related_conversation_id UUID DEFAULT NULL,
    p_action_url TEXT DEFAULT NULL,
    p_priority VARCHAR DEFAULT 'normal'
)
RETURNS UUID AS $$
DECLARE
    v_notification_id UUID;
BEGIN
    INSERT INTO notifications.notifications (
        user_id, notification_type, notification_category,
        title, body,
        related_user_id, related_message_id, related_conversation_id,
        action_url, priority
    ) VALUES (
        p_user_id, p_notification_type, 'social',
        p_title, p_body,
        p_related_user_id, p_related_message_id, p_related_conversation_id,
        p_action_url, p_priority
    ) RETURNING id INTO v_notification_id;
    RETURN v_notification_id;
END;
$$ LANGUAGE plpgsql;

-- database-dup/functions/notifications.trigger_functions.sql
CREATE OR REPLACE FUNCTION notifications.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION notifications.create_default_preferences()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO notifications.user_preferences (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION notifications.create_default_notification_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO notifications.user_stats (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
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
CREATE OR REPLACE FUNCTION notifications.increment_announcement_views()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE notifications.announcements
    SET view_count = view_count + 1
    WHERE id = NEW.announcement_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
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

-- database-dup/functions/users.functions.sql
CREATE OR REPLACE FUNCTION users.are_contacts(p_user_id UUID, p_other_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users.contacts
        WHERE user_id = p_user_id
          AND contact_user_id = p_other_user_id
          AND status = 'accepted'
    );
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.is_blocked(p_user_id UUID, p_other_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users.blocked_users
        WHERE user_id = p_user_id
          AND blocked_user_id = p_other_user_id
          AND unblocked_at IS NULL
    );
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.block_user(
    p_user_id UUID,
    p_blocked_user_id UUID,
    p_block_reason TEXT DEFAULT NULL,
    p_block_type VARCHAR DEFAULT 'full'
)
RETURNS UUID AS $$
DECLARE
    v_block_id UUID;
BEGIN
    IF users.is_blocked(p_user_id, p_blocked_user_id) THEN
        RAISE EXCEPTION 'User is already blocked';
    END IF;
    INSERT INTO users.blocked_users (
        user_id, blocked_user_id, block_reason, block_type
    ) VALUES (
        p_user_id, p_blocked_user_id, p_block_reason, p_block_type
    ) RETURNING id INTO v_block_id;
    UPDATE users.contacts
    SET relationship_type = 'blocked',
        status = 'blocked',
        blocked_at = NOW(),
        block_reason = p_block_reason
    WHERE user_id = p_user_id
      AND contact_user_id = p_blocked_user_id;
    RETURN v_block_id;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.unblock_user(p_user_id UUID, p_blocked_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE users.blocked_users
    SET unblocked_at = NOW()
    WHERE user_id = p_user_id
      AND blocked_user_id = p_blocked_user_id
      AND unblocked_at IS NULL;
    UPDATE users.contacts
    SET relationship_type = 'contact',
        status = 'accepted'
    WHERE user_id = p_user_id
      AND contact_user_id = p_blocked_user_id
      AND relationship_type = 'blocked';
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.update_online_status(
    p_user_id UUID,
    p_status VARCHAR,
    p_last_seen TIMESTAMPTZ DEFAULT NOW()
)
RETURNS VOID AS $$
BEGIN
    UPDATE users.profiles
    SET online_status = p_status,
        last_seen_at = p_last_seen,
        updated_at = NOW()
    WHERE user_id = p_user_id;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.set_inactive_users_offline(p_minutes INTEGER DEFAULT 5)
RETURNS INTEGER AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    UPDATE users.profiles
    SET online_status = 'offline'
    WHERE online_status IN ('online', 'away')
      AND last_seen_at < NOW() - (p_minutes || ' minutes')::INTERVAL;
    GET DIAGNOSTICS v_updated_count = ROW_COUNT;
    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.cleanup_expired_statuses()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    UPDATE users.status_history
    SET deleted_at = NOW()
    WHERE expires_at < NOW()
      AND deleted_at IS NULL;
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.get_user_contacts(
    p_user_id UUID,
    p_relationship_type VARCHAR DEFAULT NULL
)
RETURNS TABLE (
    contact_id UUID,
    username VARCHAR,
    display_name VARCHAR,
    avatar_url TEXT,
    online_status VARCHAR,
    last_seen_at TIMESTAMPTZ,
    is_favorite BOOLEAN,
    is_pinned BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.contact_user_id,
        p.username,
        p.display_name,
        p.avatar_url,
        p.online_status,
        p.last_seen_at,
        c.is_favorite,
        c.is_pinned
    FROM users.contacts c
    JOIN users.profiles p ON p.user_id = c.contact_user_id
    WHERE c.user_id = p_user_id
      AND c.status = 'accepted'
      AND (p_relationship_type IS NULL OR c.relationship_type = p_relationship_type)
    ORDER BY c.is_pinned DESC, c.is_favorite DESC, p.display_name ASC;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.search_users(
    p_search_term TEXT,
    p_limit INTEGER DEFAULT 20
)
RETURNS TABLE (
    user_id UUID,
    username VARCHAR,
    display_name VARCHAR,
    avatar_url TEXT,
    bio TEXT,
    is_verified BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        p.user_id,
        p.username,
        p.display_name,
        p.avatar_url,
        p.bio,
        p.is_verified
    FROM users.profiles p
    WHERE p.search_visibility = TRUE
      AND p.deactivated_at IS NULL
      AND (
          p.username ILIKE '%' || p_search_term || '%'
          OR p.display_name ILIKE '%' || p_search_term || '%'
          OR to_tsvector('english', COALESCE(p.display_name, '') || ' ' || COALESCE(p.username, '') || ' ' || COALESCE(p.bio, ''))
             @@ plainto_tsquery('english', p_search_term)
      )
    ORDER BY
        CASE WHEN p.username = p_search_term THEN 0 ELSE 1 END,
        p.is_verified DESC,
        p.display_name
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.log_activity(
    p_user_id UUID,
    p_activity_type VARCHAR,
    p_activity_category VARCHAR,
    p_description TEXT,
    p_old_value JSONB DEFAULT NULL,
    p_new_value JSONB DEFAULT NULL,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_device_id VARCHAR DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_log_id UUID;
BEGIN
    INSERT INTO users.activity_log (
        user_id, activity_type, activity_category, description,
        old_value, new_value, ip_address, user_agent, device_id
    ) VALUES (
        p_user_id, p_activity_type, p_activity_category, p_description,
        p_old_value, p_new_value, p_ip_address, p_user_agent, p_device_id
    ) RETURNING id INTO v_log_id;
    RETURN v_log_id;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.update_contact_group_count(p_group_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE users.contact_groups
    SET member_count = (
        SELECT COUNT(*)
        FROM users.contacts
        WHERE p_group_id = ANY(contact_groups)
          AND user_id = users.contact_groups.user_id
    )
    WHERE id = p_group_id;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.register_device(
    p_user_id UUID,
    p_device_id VARCHAR,
    p_device_name VARCHAR,
    p_device_type VARCHAR,
    p_device_model VARCHAR,
    p_device_manufacturer VARCHAR,
    p_os_name VARCHAR,
    p_os_version VARCHAR,
    p_app_version VARCHAR,
    p_fcm_token TEXT DEFAULT NULL,
    p_apns_token TEXT DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_device_record_id UUID;
BEGIN
    UPDATE users.devices
    SET is_current_device = FALSE
    WHERE user_id = p_user_id;
    INSERT INTO users.devices (
        user_id, device_id, device_name, device_type, device_model,
        device_manufacturer, os_name, os_version, app_version,
        fcm_token, apns_token, is_current_device, last_active_at
    ) VALUES (
        p_user_id, p_device_id, p_device_name, p_device_type, p_device_model,
        p_device_manufacturer, p_os_name, p_os_version, p_app_version,
        p_fcm_token, p_apns_token, TRUE, NOW()
    )
    ON CONFLICT (user_id, device_id) DO UPDATE SET
        device_name = EXCLUDED.device_name,
        device_type = EXCLUDED.device_type,
        is_current_device = TRUE,
        last_active_at = NOW(),
        fcm_token = COALESCE(EXCLUDED.fcm_token, users.devices.fcm_token),
        apns_token = COALESCE(EXCLUDED.apns_token, users.devices.apns_token),
        is_active = TRUE
    RETURNING id INTO v_device_record_id;
    RETURN v_device_record_id;
END;
$$ LANGUAGE plpgsql;

-- database-dup/functions/users.trigger_functions.sql
CREATE OR REPLACE FUNCTION users.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.create_default_profile()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users.profiles (user_id, username, display_name)
    VALUES (NEW.id, 'user_' || substring(NEW.id::text from 1 for 8), 'User');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.create_device_from_session()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users.devices (
        user_id,
        device_id,
        device_name,
        device_type,
        device_model,
        device_manufacturer,
        os_name,
        os_version,
        fcm_token,
        apns_token,
        push_enabled,
        is_active,
        is_current_device,
        last_active_at
    ) VALUES (
        NEW.user_id,
        COALESCE(NEW.device_id, NEW.id::TEXT),
        NEW.device_name,
        NEW.device_type,
        NEW.device_model,
        NEW.device_manufacturer,
        NEW.device_os,
        NEW.device_os_version,
        NEW.fcm_token,
        NEW.apns_token,
        NEW.push_enabled,
        TRUE,
        TRUE,
        NOW()
    )
    ON CONFLICT (user_id, device_id) DO UPDATE SET
        device_name         = EXCLUDED.device_name,
        device_type         = EXCLUDED.device_type,
        device_model        = EXCLUDED.device_model,
        device_manufacturer = EXCLUDED.device_manufacturer,
        os_name             = EXCLUDED.os_name,
        os_version          = EXCLUDED.os_version,
        fcm_token           = EXCLUDED.fcm_token,
        apns_token          = EXCLUDED.apns_token,
        push_enabled        = EXCLUDED.push_enabled,
        is_active           = TRUE,
        is_current_device   = TRUE,
        last_active_at      = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.create_default_settings()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users.settings (user_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.validate_username()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.username !~ '^[a-zA-Z0-9_]{3,50}$' THEN
        RAISE EXCEPTION 'Username must be 3-50 alphanumeric characters or underscores';
    END IF;
    NEW.username := LOWER(NEW.username);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.prevent_self_contact()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.user_id = NEW.contact_user_id THEN
        RAISE EXCEPTION 'Cannot add yourself as a contact';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.prevent_self_blocking()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.user_id = NEW.blocked_user_id THEN
        RAISE EXCEPTION 'Cannot block yourself';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.log_profile_changes()
RETURNS TRIGGER AS $$
DECLARE
    v_changes JSONB := '{}'::JSONB;
BEGIN
    IF OLD.display_name IS DISTINCT FROM NEW.display_name THEN
        v_changes := v_changes || jsonb_build_object(
            'display_name', jsonb_build_object('old', OLD.display_name, 'new', NEW.display_name)
        );
    END IF;
    IF OLD.bio IS DISTINCT FROM NEW.bio THEN
        v_changes := v_changes || jsonb_build_object(
            'bio', jsonb_build_object('old', OLD.bio, 'new', NEW.bio)
        );
    END IF;
    IF OLD.avatar_url IS DISTINCT FROM NEW.avatar_url THEN
        v_changes := v_changes || jsonb_build_object(
            'avatar_url', jsonb_build_object('old', OLD.avatar_url, 'new', NEW.avatar_url)
        );
    END IF;
    IF v_changes != '{}'::JSONB THEN
        PERFORM users.log_activity(
            NEW.user_id,
            'profile_update',
            'profile',
            'User updated their profile',
            jsonb_build_object('old', v_changes),
            jsonb_build_object('new', v_changes),
            NULL, NULL, NULL
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.update_contact_accepted_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'accepted' AND OLD.status != 'accepted' THEN
        NEW.accepted_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.update_contact_interaction()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users.contacts
    SET last_interaction_at = NOW(),
        interaction_count = interaction_count + 1
    WHERE user_id = NEW.user_id
      AND contact_user_id = NEW.contact_user_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.increment_status_views()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users.status_history
    SET views_count = views_count + 1
    WHERE id = NEW.status_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.update_device_last_active()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_active_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION users.cleanup_old_devices()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users.devices
    SET is_active = FALSE
    WHERE user_id = NEW.user_id
      AND last_active_at < NOW() - INTERVAL '90 days'
      AND is_active = TRUE;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- database-dup/indexes/auth.indexes.sql
CREATE INDEX idx_auth_users_email ON auth.users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_auth_users_phone ON auth.users(phone_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_auth_users_status ON auth.users(account_status);
CREATE INDEX idx_auth_users_created ON auth.users(created_at);
CREATE INDEX idx_auth_users_deleted ON auth.users(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_auth_users_2fa ON auth.users(two_factor_enabled) WHERE two_factor_enabled = TRUE;
CREATE INDEX idx_auth_users_locked ON auth.users(account_locked_until) WHERE account_locked_until IS NOT NULL;
CREATE INDEX idx_auth_sessions_user ON auth.sessions(user_id);
CREATE INDEX idx_auth_sessions_token ON auth.sessions(session_token);
CREATE INDEX idx_auth_sessions_refresh_token ON auth.sessions(refresh_token);
CREATE INDEX idx_auth_sessions_device ON auth.sessions(device_id);
CREATE INDEX idx_auth_sessions_expires ON auth.sessions(expires_at);
CREATE INDEX idx_auth_sessions_active ON auth.sessions(user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX idx_auth_sessions_ip ON auth.sessions(ip_address);
CREATE INDEX idx_auth_sessions_created ON auth.sessions(created_at);
CREATE INDEX idx_auth_sessions_last_activity ON auth.sessions(last_activity_at);
CREATE INDEX idx_auth_otp_identifier ON auth.otp_verifications(identifier, identifier_type);
CREATE INDEX idx_auth_otp_user ON auth.otp_verifications(user_id);
CREATE INDEX idx_auth_otp_expires ON auth.otp_verifications(expires_at);
CREATE INDEX idx_auth_otp_purpose ON auth.otp_verifications(purpose);
CREATE INDEX idx_auth_otp_verified ON auth.otp_verifications(is_verified);
CREATE INDEX idx_auth_otp_created ON auth.otp_verifications(created_at);
CREATE INDEX idx_auth_oauth_user ON auth.oauth_providers(user_id);
CREATE INDEX idx_auth_oauth_provider ON auth.oauth_providers(provider);
CREATE INDEX idx_auth_oauth_provider_user ON auth.oauth_providers(provider, provider_user_id);
CREATE INDEX idx_auth_oauth_email ON auth.oauth_providers(provider_email);
CREATE INDEX idx_auth_oauth_primary ON auth.oauth_providers(user_id, is_primary) WHERE is_primary = TRUE;
CREATE INDEX idx_auth_oauth_linked ON auth.oauth_providers(linked_at);
CREATE INDEX idx_auth_pwd_reset_user ON auth.password_reset_tokens(user_id);
CREATE INDEX idx_auth_pwd_reset_token ON auth.password_reset_tokens(token);
CREATE INDEX idx_auth_pwd_reset_expires ON auth.password_reset_tokens(expires_at);
CREATE INDEX idx_auth_pwd_reset_used ON auth.password_reset_tokens(used_at) WHERE used_at IS NULL;
CREATE INDEX idx_auth_pwd_reset_created ON auth.password_reset_tokens(created_at);
CREATE INDEX idx_auth_email_verify_user ON auth.email_verification_tokens(user_id);
CREATE INDEX idx_auth_email_verify_email ON auth.email_verification_tokens(email);
CREATE INDEX idx_auth_email_verify_token ON auth.email_verification_tokens(token);
CREATE INDEX idx_auth_email_verify_expires ON auth.email_verification_tokens(expires_at);
CREATE INDEX idx_auth_email_verify_verified ON auth.email_verification_tokens(verified_at) WHERE verified_at IS NULL;
CREATE INDEX idx_auth_security_events_user ON auth.security_events(user_id);
CREATE INDEX idx_auth_security_events_session ON auth.security_events(session_id);
CREATE INDEX idx_auth_security_events_type ON auth.security_events(event_type);
CREATE INDEX idx_auth_security_events_category ON auth.security_events(event_category);
CREATE INDEX idx_auth_security_events_severity ON auth.security_events(severity);
CREATE INDEX idx_auth_security_events_created ON auth.security_events(created_at);
CREATE INDEX idx_auth_security_events_suspicious ON auth.security_events(is_suspicious) WHERE is_suspicious = TRUE;
CREATE INDEX idx_auth_security_events_ip ON auth.security_events(ip_address);
CREATE INDEX idx_auth_login_history_user ON auth.login_history(user_id);
CREATE INDEX idx_auth_login_history_session ON auth.login_history(session_id);
CREATE INDEX idx_auth_login_history_method ON auth.login_history(login_method);
CREATE INDEX idx_auth_login_history_status ON auth.login_history(status);
CREATE INDEX idx_auth_login_history_created ON auth.login_history(created_at);
CREATE INDEX idx_auth_login_history_ip ON auth.login_history(ip_address);
CREATE INDEX idx_auth_login_history_device ON auth.login_history(device_id);
CREATE INDEX idx_auth_login_history_new_device ON auth.login_history(user_id, is_new_device) WHERE is_new_device = TRUE;
CREATE INDEX idx_auth_api_keys_user ON auth.api_keys(user_id);
CREATE INDEX idx_auth_api_keys_hash ON auth.api_keys(key_hash);
CREATE INDEX idx_auth_api_keys_prefix ON auth.api_keys(key_prefix);
CREATE INDEX idx_auth_api_keys_active ON auth.api_keys(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_auth_api_keys_expires ON auth.api_keys(expires_at);
CREATE INDEX idx_auth_api_keys_last_used ON auth.api_keys(last_used_at);
CREATE INDEX idx_auth_api_keys_service ON auth.api_keys(service_name) WHERE service_name IS NOT NULL;
-- database-dup/indexes/media.indexes.sql
CREATE INDEX idx_media_files_uploader ON media.files(uploader_user_id);
CREATE INDEX idx_media_files_type ON media.files(file_type);
CREATE INDEX idx_media_files_category ON media.files(file_category);
CREATE INDEX idx_media_files_created ON media.files(created_at);
CREATE INDEX idx_media_files_uploaded ON media.files(uploaded_at);
CREATE INDEX idx_media_files_hash ON media.files(content_hash) WHERE content_hash IS NOT NULL;
CREATE INDEX idx_media_files_checksum ON media.files(checksum) WHERE checksum IS NOT NULL;
CREATE INDEX idx_media_files_processing ON media.files(processing_status);
CREATE INDEX idx_media_files_moderation ON media.files(moderation_status);
CREATE INDEX idx_media_files_nsfw ON media.files(is_nsfw) WHERE is_nsfw = TRUE;
CREATE INDEX idx_media_files_visibility ON media.files(visibility);
CREATE INDEX idx_media_files_access_token ON media.files(access_token) WHERE access_token IS NOT NULL;
CREATE INDEX idx_media_files_expires ON media.files(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_media_files_deleted ON media.files(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_media_files_permanently_delete ON media.files(permanently_delete_at) WHERE permanently_delete_at IS NOT NULL;
CREATE INDEX idx_media_files_storage_key ON media.files(storage_key);
CREATE INDEX idx_media_files_mime_type ON media.files(mime_type);
CREATE INDEX idx_media_files_size ON media.files(file_size_bytes);
CREATE INDEX idx_media_processing_file ON media.processing_queue(file_id);
CREATE INDEX idx_media_processing_task ON media.processing_queue(task_type);
CREATE INDEX idx_media_processing_status ON media.processing_queue(status);
CREATE INDEX idx_media_processing_priority ON media.processing_queue(priority DESC, created_at);
CREATE INDEX idx_media_processing_worker ON media.processing_queue(worker_id) WHERE worker_id IS NOT NULL;
CREATE INDEX idx_media_processing_created ON media.processing_queue(created_at);
CREATE INDEX idx_media_processing_queued ON media.processing_queue(status, priority) WHERE status = 'queued';
CREATE INDEX idx_media_thumbnails_file ON media.thumbnails(file_id);
CREATE INDEX idx_media_thumbnails_size ON media.thumbnails(size_type);
CREATE INDEX idx_media_thumbnails_created ON media.thumbnails(created_at);
CREATE INDEX idx_media_transcoding_source ON media.transcoding_jobs(source_file_id);
CREATE INDEX idx_media_transcoding_output ON media.transcoding_jobs(output_file_id);
CREATE INDEX idx_media_transcoding_status ON media.transcoding_jobs(status);
CREATE INDEX idx_media_transcoding_profile ON media.transcoding_jobs(profile_name);
CREATE INDEX idx_media_transcoding_created ON media.transcoding_jobs(created_at);
CREATE INDEX idx_media_albums_user ON media.albums(user_id);
CREATE INDEX idx_media_albums_type ON media.albums(album_type);
CREATE INDEX idx_media_albums_system ON media.albums(is_system_album) WHERE is_system_album = TRUE;
CREATE INDEX idx_media_albums_visibility ON media.albums(visibility);
CREATE INDEX idx_media_albums_created ON media.albums(created_at);
CREATE INDEX idx_media_album_files_album ON media.album_files(album_id);
CREATE INDEX idx_media_album_files_file ON media.album_files(file_id);
CREATE INDEX idx_media_album_files_order ON media.album_files(album_id, display_order);
CREATE INDEX idx_media_album_files_added ON media.album_files(added_at);
CREATE INDEX idx_media_tags_user ON media.tags(user_id);
CREATE INDEX idx_media_tags_name ON media.tags(tag_name);
CREATE INDEX idx_media_tags_type ON media.tags(tag_type);
CREATE INDEX idx_media_tags_usage ON media.tags(usage_count DESC);
CREATE INDEX idx_media_file_tags_file ON media.file_tags(file_id);
CREATE INDEX idx_media_file_tags_tag ON media.file_tags(tag_id);
CREATE INDEX idx_media_file_tags_confidence ON media.file_tags(confidence_score) WHERE confidence_score IS NOT NULL;
CREATE INDEX idx_media_shares_file ON media.shares(file_id);
CREATE INDEX idx_media_shares_shared_by ON media.shares(shared_by_user_id);
CREATE INDEX idx_media_shares_shared_with_user ON media.shares(shared_with_user_id);
CREATE INDEX idx_media_shares_shared_with_conversation ON media.shares(shared_with_conversation_id);
CREATE INDEX idx_media_shares_token ON media.shares(share_token) WHERE share_token IS NOT NULL;
CREATE INDEX idx_media_shares_expires ON media.shares(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_media_shares_active ON media.shares(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_media_access_log_file ON media.access_log(file_id);
CREATE INDEX idx_media_access_log_user ON media.access_log(user_id);
CREATE INDEX idx_media_access_log_type ON media.access_log(access_type);
CREATE INDEX idx_media_access_log_created ON media.access_log(created_at);
CREATE INDEX idx_media_access_log_ip ON media.access_log(ip_address);
CREATE INDEX idx_media_access_log_success ON media.access_log(success);
CREATE INDEX idx_media_sticker_packs_creator ON media.sticker_packs(creator_user_id);
CREATE INDEX idx_media_sticker_packs_official ON media.sticker_packs(is_official) WHERE is_official = TRUE;
CREATE INDEX idx_media_sticker_packs_public ON media.sticker_packs(is_public) WHERE is_public = TRUE;
CREATE INDEX idx_media_sticker_packs_animated ON media.sticker_packs(is_animated) WHERE is_animated = TRUE;
CREATE INDEX idx_media_sticker_packs_downloads ON media.sticker_packs(download_count DESC);
CREATE INDEX idx_media_stickers_pack ON media.stickers(sticker_pack_id);
CREATE INDEX idx_media_stickers_file ON media.stickers(file_id);
CREATE INDEX idx_media_stickers_creator ON media.stickers(creator_user_id);
CREATE INDEX idx_media_stickers_usage ON media.stickers(usage_count DESC);
CREATE INDEX idx_media_stickers_active ON media.stickers(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_media_user_sticker_packs_user ON media.user_sticker_packs(user_id);
CREATE INDEX idx_media_user_sticker_packs_pack ON media.user_sticker_packs(sticker_pack_id);
CREATE INDEX idx_media_user_sticker_packs_order ON media.user_sticker_packs(user_id, display_order);
CREATE INDEX idx_media_gifs_provider ON media.gifs(provider, provider_gif_id);
CREATE INDEX idx_media_gifs_usage ON media.gifs(usage_count DESC);
CREATE INDEX idx_media_gifs_trending ON media.gifs(is_trending) WHERE is_trending = TRUE;
CREATE INDEX idx_media_gifs_tags ON media.gifs USING GIN(tags);
CREATE INDEX idx_media_favorite_gifs_user ON media.favorite_gifs(user_id);
CREATE INDEX idx_media_favorite_gifs_gif ON media.favorite_gifs(gif_id);
CREATE INDEX idx_media_favorite_gifs_added ON media.favorite_gifs(added_at);
CREATE INDEX idx_media_storage_stats_user ON media.storage_stats(user_id);
CREATE INDEX idx_media_storage_stats_usage ON media.storage_stats(storage_used_percentage DESC);
CREATE INDEX idx_media_storage_stats_calculated ON media.storage_stats(last_calculated_at);
-- database-dup/indexes/messages.indexes.sql
CREATE INDEX IF NOT EXISTS idx_messages_conversations_creator ON messages.conversations(creator_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_conversations_type ON messages.conversations(conversation_type);
CREATE INDEX IF NOT EXISTS idx_messages_conversations_last_message ON messages.conversations(last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_conversations_last_activity ON messages.conversations(last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_conversations_active ON messages.conversations(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_conversations_archived ON messages.conversations(is_archived) WHERE is_archived = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_conversations_public ON messages.conversations(is_public) WHERE is_public = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_conversations_invite ON messages.conversations(invite_link) WHERE invite_link IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_conversations_created ON messages.conversations(created_at);
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
CREATE INDEX IF NOT EXISTS idx_messages_reactions_message ON messages.reactions(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_reactions_user ON messages.reactions(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_reactions_type ON messages.reactions(reaction_type);
CREATE INDEX IF NOT EXISTS idx_messages_reactions_created ON messages.reactions(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_delivery_message ON messages.delivery_status(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_delivery_user ON messages.delivery_status(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_delivery_status ON messages.delivery_status(status);
CREATE INDEX IF NOT EXISTS idx_messages_delivery_undelivered ON messages.delivery_status(message_id, status) 
    WHERE status IN ('sent', 'failed');
CREATE INDEX IF NOT EXISTS idx_messages_delivery_unread ON messages.delivery_status(user_id, status) WHERE status = 'sent';
CREATE INDEX IF NOT EXISTS idx_messages_media_message ON messages.message_media(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_media_type ON messages.message_media(media_type);
CREATE INDEX IF NOT EXISTS idx_messages_media_order ON messages.message_media(message_id, display_order);
CREATE INDEX IF NOT EXISTS idx_messages_link_previews_message ON messages.link_previews(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_link_previews_url ON messages.link_previews(url);
CREATE INDEX IF NOT EXISTS idx_messages_polls_message ON messages.polls(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_polls_closed ON messages.polls(is_closed);
CREATE INDEX IF NOT EXISTS idx_messages_polls_closes_at ON messages.polls(closes_at) WHERE closes_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_poll_options_poll ON messages.poll_options(poll_id);
CREATE INDEX IF NOT EXISTS idx_messages_poll_options_order ON messages.poll_options(poll_id, option_order);
CREATE INDEX IF NOT EXISTS idx_messages_poll_votes_poll ON messages.poll_votes(poll_id);
CREATE INDEX IF NOT EXISTS idx_messages_poll_votes_option ON messages.poll_votes(poll_option_id);
CREATE INDEX IF NOT EXISTS idx_messages_poll_votes_user ON messages.poll_votes(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_poll_votes_voted_at ON messages.poll_votes(voted_at);
CREATE INDEX IF NOT EXISTS idx_messages_typing_conversation ON messages.typing_indicators(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_typing_user ON messages.typing_indicators(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_typing_expires ON messages.typing_indicators(expires_at);
CREATE INDEX IF NOT EXISTS idx_messages_typing_active ON messages.typing_indicators(conversation_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_messages_reports_message ON messages.message_reports(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_reports_reporter ON messages.message_reports(reporter_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_reports_type ON messages.message_reports(report_type);
CREATE INDEX IF NOT EXISTS idx_messages_reports_status ON messages.message_reports(status);
CREATE INDEX IF NOT EXISTS idx_messages_reports_priority ON messages.message_reports(priority);
CREATE INDEX IF NOT EXISTS idx_messages_reports_assigned ON messages.message_reports(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_reports_created ON messages.message_reports(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_drafts_user ON messages.drafts(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_drafts_conversation ON messages.drafts(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_drafts_updated ON messages.drafts(updated_at);
CREATE INDEX IF NOT EXISTS idx_messages_bookmarks_user ON messages.bookmarks(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_bookmarks_message ON messages.bookmarks(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_bookmarks_collection ON messages.bookmarks(user_id, collection_name);
CREATE INDEX IF NOT EXISTS idx_messages_bookmarks_bookmarked_at ON messages.bookmarks(bookmarked_at);
CREATE INDEX IF NOT EXISTS idx_messages_pinned_conversation ON messages.pinned_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_pinned_message ON messages.pinned_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_pinned_order ON messages.pinned_messages(conversation_id, pin_order);
CREATE INDEX IF NOT EXISTS idx_messages_invites_conversation ON messages.conversation_invites(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_invites_inviter ON messages.conversation_invites(inviter_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_invites_invitee ON messages.conversation_invites(invitee_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_invites_code ON messages.conversation_invites(invite_code);
CREATE INDEX IF NOT EXISTS idx_messages_invites_status ON messages.conversation_invites(status);
CREATE INDEX IF NOT EXISTS idx_messages_invites_expires ON messages.conversation_invites(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_search_message ON messages.search_index(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_search_conversation ON messages.search_index(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_search_user ON messages.search_index(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_search_content ON messages.search_index USING GIN(content_tsvector);
CREATE INDEX IF NOT EXISTS idx_messages_search_created ON messages.search_index(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_calls_conversation ON messages.calls(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_calls_initiator ON messages.calls(initiator_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_calls_type ON messages.calls(call_type);
CREATE INDEX IF NOT EXISTS idx_messages_calls_status ON messages.calls(status);
CREATE INDEX IF NOT EXISTS idx_messages_calls_started ON messages.calls(started_at);
CREATE INDEX IF NOT EXISTS idx_messages_calls_created ON messages.calls(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_call_participants_call ON messages.call_participants(call_id);
CREATE INDEX IF NOT EXISTS idx_messages_call_participants_user ON messages.call_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_call_participants_status ON messages.call_participants(status);
CREATE INDEX IF NOT EXISTS idx_messages_call_participants_joined ON messages.call_participants(joined_at);
CREATE INDEX IF NOT EXISTS idx_messages_settings_conversation ON messages.conversation_settings(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_settings_disappearing ON messages.conversation_settings(disappearing_messages_enabled) 
    WHERE disappearing_messages_enabled = TRUE;
-- database-dup/indexes/notifications.indexes.sql
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications.notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications.notifications(notification_type);
CREATE INDEX IF NOT EXISTS idx_notifications_category ON notifications.notifications(notification_category);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications.notifications(is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_seen ON notifications.notifications(is_seen);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications.notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_unseen ON notifications.notifications(user_id, is_seen) WHERE is_seen = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications.notifications(delivery_status);
CREATE INDEX IF NOT EXISTS idx_notifications_priority ON notifications.notifications(priority);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications.notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_scheduled ON notifications.notifications(scheduled_for) 
    WHERE scheduled_for IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_expires ON notifications.notifications(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_group ON notifications.notifications(group_key) WHERE group_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_deleted ON notifications.notifications(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_related_user ON notifications.notifications(related_user_id) WHERE related_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_related_message ON notifications.notifications(related_message_id) WHERE related_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_related_conversation ON notifications.notifications(related_conversation_id) WHERE related_conversation_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_push_delivery_notification ON notifications.push_delivery_log(notification_id);
CREATE INDEX IF NOT EXISTS idx_push_delivery_user ON notifications.push_delivery_log(user_id);
CREATE INDEX IF NOT EXISTS idx_push_delivery_device ON notifications.push_delivery_log(device_id);
CREATE INDEX IF NOT EXISTS idx_push_delivery_status ON notifications.push_delivery_log(status);
CREATE INDEX IF NOT EXISTS idx_push_delivery_created ON notifications.push_delivery_log(created_at);
CREATE INDEX IF NOT EXISTS idx_push_delivery_opened ON notifications.push_delivery_log(opened_at) WHERE opened_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_email_notifications_user ON notifications.email_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_email_notifications_notification ON notifications.email_notifications(notification_id);
CREATE INDEX IF NOT EXISTS idx_email_notifications_status ON notifications.email_notifications(status);
CREATE INDEX IF NOT EXISTS idx_email_notifications_created ON notifications.email_notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_email_notifications_sent ON notifications.email_notifications(sent_at);
CREATE INDEX IF NOT EXISTS idx_email_notifications_opened ON notifications.email_notifications(opened_at) WHERE opened_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_email_notifications_bounced ON notifications.email_notifications(bounced_at) WHERE bounced_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sms_notifications_user ON notifications.sms_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_sms_notifications_notification ON notifications.sms_notifications(notification_id);
CREATE INDEX IF NOT EXISTS idx_sms_notifications_phone ON notifications.sms_notifications(phone_number);
CREATE INDEX IF NOT EXISTS idx_sms_notifications_status ON notifications.sms_notifications(status);
CREATE INDEX IF NOT EXISTS idx_sms_notifications_created ON notifications.sms_notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_user_preferences_user ON notifications.user_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_channels_user ON notifications.conversation_channels(user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_channels_conversation ON notifications.conversation_channels(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conversation_channels_muted ON notifications.conversation_channels(is_muted) WHERE is_muted = TRUE;
CREATE INDEX IF NOT EXISTS idx_templates_name ON notifications.templates(template_name);
CREATE INDEX IF NOT EXISTS idx_templates_type ON notifications.templates(template_type);
CREATE INDEX IF NOT EXISTS idx_templates_language ON notifications.templates(language_code);
CREATE INDEX IF NOT EXISTS idx_templates_active ON notifications.templates(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_notification_actions_notification ON notifications.notification_actions(notification_id);
CREATE INDEX IF NOT EXISTS idx_notification_actions_id ON notifications.notification_actions(action_id);
CREATE INDEX IF NOT EXISTS idx_notification_actions_order ON notifications.notification_actions(notification_id, display_order);
CREATE INDEX IF NOT EXISTS idx_action_responses_notification ON notifications.action_responses(notification_id);
CREATE INDEX IF NOT EXISTS idx_action_responses_user ON notifications.action_responses(user_id);
CREATE INDEX IF NOT EXISTS idx_action_responses_action ON notifications.action_responses(action_id);
CREATE INDEX IF NOT EXISTS idx_action_responses_responded ON notifications.action_responses(responded_at);
CREATE INDEX IF NOT EXISTS idx_batches_status ON notifications.batches(status);
CREATE INDEX IF NOT EXISTS idx_batches_priority ON notifications.batches(priority);
CREATE INDEX IF NOT EXISTS idx_batches_scheduled ON notifications.batches(scheduled_for) WHERE scheduled_for IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_batches_created ON notifications.batches(created_at);
CREATE INDEX IF NOT EXISTS idx_batches_created_by ON notifications.batches(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_announcements_active ON notifications.announcements(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_announcements_type ON notifications.announcements(announcement_type);
CREATE INDEX IF NOT EXISTS idx_announcements_severity ON notifications.announcements(severity);
CREATE INDEX IF NOT EXISTS idx_announcements_starts ON notifications.announcements(starts_at);
CREATE INDEX IF NOT EXISTS idx_announcements_ends ON notifications.announcements(ends_at);
CREATE INDEX IF NOT EXISTS idx_announcements_audience ON notifications.announcements(target_audience);
CREATE INDEX IF NOT EXISTS idx_announcements_priority ON notifications.announcements(display_priority);
CREATE INDEX IF NOT EXISTS idx_announcements_active_period ON notifications.announcements(is_active, starts_at, ends_at) 
    WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_announcement_views_announcement ON notifications.announcement_views(announcement_id);
CREATE INDEX IF NOT EXISTS idx_announcement_views_user ON notifications.announcement_views(user_id);
CREATE INDEX IF NOT EXISTS idx_announcement_views_first_viewed ON notifications.announcement_views(first_viewed_at);
CREATE INDEX IF NOT EXISTS idx_announcement_views_clicked ON notifications.announcement_views(clicked) WHERE clicked = TRUE;
CREATE INDEX IF NOT EXISTS idx_announcement_views_dismissed ON notifications.announcement_views(dismissed) WHERE dismissed = TRUE;
CREATE INDEX IF NOT EXISTS idx_user_stats_user ON notifications.user_stats(user_id);
CREATE INDEX IF NOT EXISTS idx_user_stats_last_notification ON notifications.user_stats(last_notification_at);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON notifications.subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_topic ON notifications.subscriptions(topic_name);
CREATE INDEX IF NOT EXISTS idx_subscriptions_subscribed ON notifications.subscriptions(is_subscribed) WHERE is_subscribed = TRUE;
-- database-dup/indexes/users.indexes.sql
CREATE INDEX IF NOT EXISTS idx_users_profiles_user ON users.profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_users_profiles_username ON users.profiles(username);
CREATE INDEX IF NOT EXISTS idx_users_profiles_display_name ON users.profiles(display_name);
CREATE INDEX IF NOT EXISTS idx_users_profiles_email_visible ON users.profiles(email_visible) WHERE email_visible = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_profiles_phone_visible ON users.profiles(phone_visible) WHERE phone_visible = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_profiles_online_status ON users.profiles(online_status);
CREATE INDEX IF NOT EXISTS idx_users_profiles_last_seen ON users.profiles(last_seen_at);
CREATE INDEX IF NOT EXISTS idx_users_profiles_visibility ON users.profiles(profile_visibility);
CREATE INDEX IF NOT EXISTS idx_users_profiles_search ON users.profiles(search_visibility) WHERE search_visibility = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_profiles_verified ON users.profiles(is_verified) WHERE is_verified = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_profiles_country ON users.profiles(country_code);
CREATE INDEX IF NOT EXISTS idx_users_profiles_city ON users.profiles(city);
CREATE INDEX IF NOT EXISTS idx_users_profiles_created ON users.profiles(created_at);
CREATE INDEX IF NOT EXISTS idx_users_profiles_deactivated ON users.profiles(deactivated_at) WHERE deactivated_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_profiles_search_text ON users.profiles 
    USING GIN(to_tsvector('english', COALESCE(display_name, '') || ' ' || COALESCE(username, '') || ' ' || COALESCE(bio, '')));
CREATE INDEX IF NOT EXISTS idx_users_contacts_user ON users.contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_users_contacts_contact_user ON users.contacts(contact_user_id);
CREATE INDEX IF NOT EXISTS idx_users_contacts_relationship ON users.contacts(relationship_type);
CREATE INDEX IF NOT EXISTS idx_users_contacts_status ON users.contacts(status);
CREATE INDEX IF NOT EXISTS idx_users_contacts_favorite ON users.contacts(user_id, is_favorite) WHERE is_favorite = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_contacts_pinned ON users.contacts(user_id, is_pinned) WHERE is_pinned = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_contacts_archived ON users.contacts(is_archived);
CREATE INDEX IF NOT EXISTS idx_users_contacts_muted ON users.contacts(is_muted) WHERE is_muted = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_contacts_source ON users.contacts(contact_source);
CREATE INDEX IF NOT EXISTS idx_users_contacts_last_interaction ON users.contacts(last_interaction_at);
CREATE INDEX IF NOT EXISTS idx_users_contacts_created ON users.contacts(created_at);
CREATE INDEX IF NOT EXISTS idx_users_contact_groups_user ON users.contact_groups(user_id);
CREATE INDEX IF NOT EXISTS idx_users_contact_groups_name ON users.contact_groups(user_id, group_name);
CREATE INDEX IF NOT EXISTS idx_users_contact_groups_default ON users.contact_groups(user_id, is_default) WHERE is_default = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_settings_user ON users.settings(user_id);
CREATE INDEX IF NOT EXISTS idx_users_blocked_user ON users.blocked_users(user_id);
CREATE INDEX IF NOT EXISTS idx_users_blocked_blocked_user ON users.blocked_users(blocked_user_id);
CREATE INDEX IF NOT EXISTS idx_users_blocked_at ON users.blocked_users(blocked_at);
CREATE INDEX IF NOT EXISTS idx_users_blocked_unblocked ON users.blocked_users(unblocked_at) WHERE unblocked_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_blocked_type ON users.blocked_users(block_type);
CREATE INDEX IF NOT EXISTS idx_users_privacy_overrides_user ON users.privacy_overrides(user_id);
CREATE INDEX IF NOT EXISTS idx_users_privacy_overrides_target ON users.privacy_overrides(target_user_id);
CREATE INDEX IF NOT EXISTS idx_users_status_history_user ON users.status_history(user_id);
CREATE INDEX IF NOT EXISTS idx_users_status_history_created ON users.status_history(created_at);
CREATE INDEX IF NOT EXISTS idx_users_status_history_expires ON users.status_history(expires_at);
CREATE INDEX IF NOT EXISTS idx_users_status_history_deleted ON users.status_history(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status_history_privacy ON users.status_history(privacy);
CREATE INDEX IF NOT EXISTS idx_users_status_history_active ON users.status_history(user_id, expires_at) 
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status_views_status ON users.status_views(status_id);
CREATE INDEX IF NOT EXISTS idx_users_status_views_viewer ON users.status_views(viewer_user_id);
CREATE INDEX IF NOT EXISTS idx_users_status_views_viewed_at ON users.status_views(viewed_at);
CREATE INDEX IF NOT EXISTS idx_users_activity_log_user ON users.activity_log(user_id);
CREATE INDEX IF NOT EXISTS idx_users_activity_log_type ON users.activity_log(activity_type);
CREATE INDEX IF NOT EXISTS idx_users_activity_log_category ON users.activity_log(activity_category);
CREATE INDEX IF NOT EXISTS idx_users_activity_log_created ON users.activity_log(created_at);
CREATE INDEX IF NOT EXISTS idx_users_activity_log_ip ON users.activity_log(ip_address);
CREATE INDEX IF NOT EXISTS idx_users_preferences_user ON users.preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_users_preferences_key ON users.preferences(preference_key);
CREATE INDEX IF NOT EXISTS idx_users_preferences_category ON users.preferences(category);
CREATE INDEX IF NOT EXISTS idx_users_preferences_system ON users.preferences(is_system) WHERE is_system = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_devices_user ON users.devices(user_id);
CREATE INDEX IF NOT EXISTS idx_users_devices_device_id ON users.devices(device_id);
CREATE INDEX IF NOT EXISTS idx_users_devices_current ON users.devices(user_id, is_current_device) WHERE is_current_device = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_devices_active ON users.devices(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_devices_last_active ON users.devices(last_active_at);
CREATE INDEX IF NOT EXISTS idx_users_devices_push_enabled ON users.devices(push_enabled) WHERE push_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_devices_fcm_token ON users.devices(fcm_token) WHERE fcm_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_devices_apns_token ON users.devices(apns_token) WHERE apns_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_achievements_user ON users.achievements(user_id);
CREATE INDEX IF NOT EXISTS idx_users_achievements_type ON users.achievements(achievement_type);
CREATE INDEX IF NOT EXISTS idx_users_achievements_unlocked ON users.achievements(is_unlocked) WHERE is_unlocked = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_achievements_display ON users.achievements(user_id, display_on_profile) WHERE display_on_profile = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_achievements_rarity ON users.achievements(achievement_rarity);
CREATE INDEX IF NOT EXISTS idx_users_reports_reporter ON users.reports(reporter_user_id);
CREATE INDEX IF NOT EXISTS idx_users_reports_reported ON users.reports(reported_user_id);
CREATE INDEX IF NOT EXISTS idx_users_reports_type ON users.reports(report_type);
CREATE INDEX IF NOT EXISTS idx_users_reports_status ON users.reports(status);
CREATE INDEX IF NOT EXISTS idx_users_reports_priority ON users.reports(priority);
CREATE INDEX IF NOT EXISTS idx_users_reports_assigned ON users.reports(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_reports_created ON users.reports(created_at);
CREATE INDEX IF NOT EXISTS idx_users_reports_resolved ON users.reports(resolved_at) WHERE resolved_at IS NOT NULL;
-- database-dup/rls/auth.rls.sql
ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.otp_verifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.oauth_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.password_reset_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.email_verification_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.security_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.login_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.api_keys ENABLE ROW LEVEL SECURITY;
CREATE OR REPLACE FUNCTION auth.current_user_id()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_user_id', TRUE)::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
CREATE OR REPLACE FUNCTION auth.is_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM auth.users
        WHERE id = auth.current_user_id()
        AND account_status = 'active'
        AND metadata->>'role' = 'admin'
    );
EXCEPTION
    WHEN OTHERS THEN
        RETURN FALSE;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
DROP POLICY IF EXISTS users_select_own ON auth.users;
CREATE POLICY users_select_own
    ON auth.users
    FOR SELECT
    USING (id = auth.current_user_id());
DROP POLICY IF EXISTS users_select_admin ON auth.users;
CREATE POLICY users_select_admin
    ON auth.users
    FOR SELECT
    USING (auth.is_admin());
DROP POLICY IF EXISTS users_update_own ON auth.users;
CREATE POLICY users_update_own
    ON auth.users
    FOR UPDATE
    USING (id = auth.current_user_id())
    WITH CHECK (id = auth.current_user_id());
DROP POLICY IF EXISTS users_update_admin ON auth.users;
CREATE POLICY users_update_admin
    ON auth.users
    FOR UPDATE
    USING (auth.is_admin());
DROP POLICY IF EXISTS users_insert_service ON auth.users;
CREATE POLICY users_insert_service
    ON auth.users
    FOR INSERT
    WITH CHECK (TRUE); 
DROP POLICY IF EXISTS users_delete_admin ON auth.users;
CREATE POLICY users_delete_admin
    ON auth.users
    FOR DELETE
    USING (auth.is_admin());
DROP POLICY IF EXISTS sessions_select_own ON auth.sessions;
CREATE POLICY sessions_select_own
    ON auth.sessions
    FOR SELECT
    USING (user_id = auth.current_user_id());
DROP POLICY IF EXISTS sessions_update_own ON auth.sessions;
CREATE POLICY sessions_update_own
    ON auth.sessions
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
DROP POLICY IF EXISTS sessions_insert_service ON auth.sessions;
CREATE POLICY sessions_insert_service
    ON auth.sessions
    FOR INSERT
    WITH CHECK (TRUE);
DROP POLICY IF EXISTS sessions_delete_own ON auth.sessions;
CREATE POLICY sessions_delete_own
    ON auth.sessions
    FOR DELETE
    USING (user_id = auth.current_user_id());
DROP POLICY IF EXISTS sessions_admin_all ON auth.sessions;
CREATE POLICY sessions_admin_all
    ON auth.sessions
    FOR ALL
    USING (auth.is_admin());
DROP POLICY IF EXISTS otp_service_all ON auth.otp_verifications;
CREATE POLICY otp_service_all
    ON auth.otp_verifications
    FOR ALL
    USING (TRUE);
DROP POLICY IF EXISTS otp_select_own ON auth.otp_verifications;
CREATE POLICY otp_select_own
    ON auth.otp_verifications
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR identifier IN (
            SELECT email FROM auth.users WHERE id = auth.current_user_id()
            UNION
            SELECT phone_number FROM auth.users WHERE id = auth.current_user_id()
        )
    );
DROP POLICY IF EXISTS oauth_select_own ON auth.oauth_providers;
CREATE POLICY oauth_select_own
    ON auth.oauth_providers
    FOR SELECT
    USING (user_id = auth.current_user_id());
DROP POLICY IF EXISTS oauth_delete_own ON auth.oauth_providers;
CREATE POLICY oauth_delete_own
    ON auth.oauth_providers
    FOR DELETE
    USING (user_id = auth.current_user_id());
DROP POLICY IF EXISTS oauth_service_all ON auth.oauth_providers;
CREATE POLICY oauth_service_all
    ON auth.oauth_providers
    FOR ALL
    USING (TRUE);
DROP POLICY IF EXISTS password_reset_service_all ON auth.password_reset_tokens;
CREATE POLICY password_reset_service_all
    ON auth.password_reset_tokens
    FOR ALL
    USING (TRUE);
DROP POLICY IF EXISTS password_reset_select_own ON auth.password_reset_tokens;
CREATE POLICY password_reset_select_own
    ON auth.password_reset_tokens
    FOR SELECT
    USING (user_id = auth.current_user_id());
DROP POLICY IF EXISTS email_verify_service_all ON auth.email_verification_tokens;
CREATE POLICY email_verify_service_all
    ON auth.email_verification_tokens
    FOR ALL
    USING (TRUE);
DROP POLICY IF EXISTS security_events_select_own ON auth.security_events;
CREATE POLICY security_events_select_own
    ON auth.security_events
    FOR SELECT
    USING (user_id = auth.current_user_id());
DROP POLICY IF EXISTS security_events_select_admin ON auth.security_events;
CREATE POLICY security_events_select_admin
    ON auth.security_events
    FOR SELECT
    USING (auth.is_admin());
DROP POLICY IF EXISTS security_events_insert_service ON auth.security_events;
CREATE POLICY security_events_insert_service
    ON auth.security_events
    FOR INSERT
    WITH CHECK (TRUE);
DROP POLICY IF EXISTS login_history_select_own ON auth.login_history;
CREATE POLICY login_history_select_own
    ON auth.login_history
    FOR SELECT
    USING (user_id = auth.current_user_id());
DROP POLICY IF EXISTS login_history_select_admin ON auth.login_history;
CREATE POLICY login_history_select_admin
    ON auth.login_history
    FOR SELECT
    USING (auth.is_admin());
DROP POLICY IF EXISTS login_history_insert_service ON auth.login_history;
CREATE POLICY login_history_insert_service
    ON auth.login_history
    FOR INSERT
    WITH CHECK (TRUE);
DROP POLICY IF EXISTS api_keys_select_own ON auth.api_keys;
CREATE POLICY api_keys_select_own
    ON auth.api_keys
    FOR SELECT
    USING (user_id = auth.current_user_id());
DROP POLICY IF EXISTS api_keys_insert_own ON auth.api_keys;
CREATE POLICY api_keys_insert_own
    ON auth.api_keys
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());
DROP POLICY IF EXISTS api_keys_update_own ON auth.api_keys;
CREATE POLICY api_keys_update_own
    ON auth.api_keys
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
DROP POLICY IF EXISTS api_keys_delete_own ON auth.api_keys;
CREATE POLICY api_keys_delete_own
    ON auth.api_keys
    FOR DELETE
    USING (user_id = auth.current_user_id());
DROP POLICY IF EXISTS api_keys_admin_all ON auth.api_keys;
CREATE POLICY api_keys_admin_all
    ON auth.api_keys
    FOR ALL
    USING (auth.is_admin());
DROP POLICY IF EXISTS api_keys_service_own ON auth.api_keys;
CREATE POLICY api_keys_service_own
    ON auth.api_keys
    FOR SELECT
    USING (service_name IS NOT NULL);
-- database-dup/rls/media.rls.sql
ALTER TABLE media.files ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.processing_queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.thumbnails ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.transcoding_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.albums ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.album_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.file_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.shares ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.access_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.sticker_packs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.stickers ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.user_sticker_packs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.gifs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.favorite_gifs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.storage_stats ENABLE ROW LEVEL SECURITY;
CREATE POLICY files_select_own
    ON media.files
    FOR SELECT
    USING (uploader_user_id = auth.current_user_id() AND deleted_at IS NULL);
CREATE POLICY files_select_public
    ON media.files
    FOR SELECT
    USING (visibility = 'public' AND deleted_at IS NULL);
CREATE POLICY files_select_shared
    ON media.files
    FOR SELECT
    USING (
        deleted_at IS NULL
        AND EXISTS (
            SELECT 1 FROM media.shares s
            WHERE s.file_id = media.files.id
            AND (
                s.shared_with_user_id = auth.current_user_id()
                OR EXISTS (
                    SELECT 1 FROM messages.conversation_participants cp
                    WHERE cp.conversation_id = s.shared_with_conversation_id
                    AND cp.user_id = auth.current_user_id()
                    AND cp.left_at IS NULL
                )
            )
            AND s.is_active = TRUE
            AND (s.expires_at IS NULL OR s.expires_at > NOW())
        )
    );
CREATE POLICY files_insert_own
    ON media.files
    FOR INSERT
    WITH CHECK (uploader_user_id = auth.current_user_id());
CREATE POLICY files_update_own
    ON media.files
    FOR UPDATE
    USING (uploader_user_id = auth.current_user_id())
    WITH CHECK (uploader_user_id = auth.current_user_id());
CREATE POLICY files_delete_own
    ON media.files
    FOR DELETE
    USING (uploader_user_id = auth.current_user_id());
CREATE POLICY files_admin_all
    ON media.files
    FOR ALL
    USING (auth.is_admin());
CREATE POLICY processing_queue_service_all
    ON media.processing_queue
    FOR ALL
    USING (TRUE);
CREATE POLICY thumbnails_select
    ON media.thumbnails
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.thumbnails.file_id
            AND (
                f.uploader_user_id = auth.current_user_id()
                OR f.visibility = 'public'
            )
        )
    );
CREATE POLICY thumbnails_service_all
    ON media.thumbnails
    FOR ALL
    USING (TRUE);
CREATE POLICY transcoding_jobs_service_all
    ON media.transcoding_jobs
    FOR ALL
    USING (TRUE);
CREATE POLICY albums_select_own
    ON media.albums
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY albums_select_public
    ON media.albums
    FOR SELECT
    USING (visibility = 'public');
CREATE POLICY albums_insert_own
    ON media.albums
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY albums_update_own
    ON media.albums
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY albums_delete_own
    ON media.albums
    FOR DELETE
    USING (user_id = auth.current_user_id());
CREATE POLICY album_files_all_own
    ON media.album_files
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM media.albums a
            WHERE a.id = media.album_files.album_id
            AND a.user_id = auth.current_user_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM media.albums a
            WHERE a.id = media.album_files.album_id
            AND a.user_id = auth.current_user_id()
        )
    );
CREATE POLICY tags_select_all
    ON media.tags
    FOR SELECT
    USING (TRUE);
CREATE POLICY tags_insert_own
    ON media.tags
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id() OR user_id IS NULL);
CREATE POLICY file_tags_select
    ON media.file_tags
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.file_tags.file_id
            AND (
                f.uploader_user_id = auth.current_user_id()
                OR f.visibility = 'public'
            )
        )
    );
CREATE POLICY file_tags_insert_own
    ON media.file_tags
    FOR INSERT
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.file_tags.file_id
            AND f.uploader_user_id = auth.current_user_id()
        )
    );
CREATE POLICY file_tags_delete_own
    ON media.file_tags
    FOR DELETE
    USING (
        EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.file_tags.file_id
            AND f.uploader_user_id = auth.current_user_id()
        )
    );
CREATE POLICY shares_select_own
    ON media.shares
    FOR SELECT
    USING (shared_by_user_id = auth.current_user_id());
CREATE POLICY shares_select_shared_with
    ON media.shares
    FOR SELECT
    USING (
        shared_with_user_id = auth.current_user_id()
        OR EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = media.shares.shared_with_conversation_id
            AND cp.user_id = auth.current_user_id()
        )
    );
CREATE POLICY shares_insert_own
    ON media.shares
    FOR INSERT
    WITH CHECK (
        shared_by_user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.shares.file_id
            AND f.uploader_user_id = auth.current_user_id()
        )
    );
CREATE POLICY shares_update_own
    ON media.shares
    FOR UPDATE
    USING (shared_by_user_id = auth.current_user_id())
    WITH CHECK (shared_by_user_id = auth.current_user_id());
CREATE POLICY shares_delete_own
    ON media.shares
    FOR DELETE
    USING (shared_by_user_id = auth.current_user_id());
CREATE POLICY access_log_select_own
    ON media.access_log
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.access_log.file_id
            AND f.uploader_user_id = auth.current_user_id()
        )
    );
CREATE POLICY access_log_insert_service
    ON media.access_log
    FOR INSERT
    WITH CHECK (TRUE);
CREATE POLICY sticker_packs_select_public
    ON media.sticker_packs
    FOR SELECT
    USING (is_public = TRUE);
CREATE POLICY sticker_packs_select_own
    ON media.sticker_packs
    FOR SELECT
    USING (creator_user_id = auth.current_user_id());
CREATE POLICY sticker_packs_insert_own
    ON media.sticker_packs
    FOR INSERT
    WITH CHECK (creator_user_id = auth.current_user_id());
CREATE POLICY sticker_packs_update_own
    ON media.sticker_packs
    FOR UPDATE
    USING (creator_user_id = auth.current_user_id())
    WITH CHECK (creator_user_id = auth.current_user_id());
CREATE POLICY stickers_select
    ON media.stickers
    FOR SELECT
    USING (
        is_active = TRUE
        AND EXISTS (
            SELECT 1 FROM media.sticker_packs sp
            WHERE sp.id = media.stickers.sticker_pack_id
            AND (sp.is_public = TRUE OR sp.creator_user_id = auth.current_user_id())
        )
    );
CREATE POLICY stickers_manage_own
    ON media.stickers
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM media.sticker_packs sp
            WHERE sp.id = media.stickers.sticker_pack_id
            AND sp.creator_user_id = auth.current_user_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM media.sticker_packs sp
            WHERE sp.id = media.stickers.sticker_pack_id
            AND sp.creator_user_id = auth.current_user_id()
        )
    );
CREATE POLICY user_sticker_packs_all_own
    ON media.user_sticker_packs
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY gifs_select_all
    ON media.gifs
    FOR SELECT
    USING (TRUE);
CREATE POLICY gifs_service_all
    ON media.gifs
    FOR ALL
    USING (TRUE);
CREATE POLICY favorite_gifs_all_own
    ON media.favorite_gifs
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY storage_stats_select_own
    ON media.storage_stats
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY storage_stats_service_all
    ON media.storage_stats
    FOR ALL
    USING (TRUE);
-- database-dup/rls/messages.rls.sql
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
CREATE POLICY conversations_select_public
    ON messages.conversations
    FOR SELECT
    USING (is_public = TRUE AND is_active = TRUE);
CREATE POLICY conversations_insert_own
    ON messages.conversations
    FOR INSERT
    WITH CHECK (creator_user_id = auth.current_user_id());
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
CREATE POLICY conversations_admin_all
    ON messages.conversations
    FOR ALL
    USING (auth.is_admin());
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
        OR user_id = auth.current_user_id() 
    );
CREATE POLICY participants_update_own
    ON messages.conversation_participants
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (
        user_id = auth.current_user_id()
    );
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
CREATE POLICY participants_delete_self
    ON messages.conversation_participants
    FOR DELETE
    USING (user_id = auth.current_user_id());
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
CREATE POLICY messages_update_own
    ON messages.messages
    FOR UPDATE
    USING (sender_user_id = auth.current_user_id())
    WITH CHECK (sender_user_id = auth.current_user_id());
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
CREATE POLICY reactions_delete_own
    ON messages.reactions
    FOR DELETE
    USING (user_id = auth.current_user_id());
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
CREATE POLICY delivery_status_select_own
    ON messages.delivery_status
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY delivery_status_service_all
    ON messages.delivery_status
    FOR ALL
    USING (TRUE);
CREATE POLICY delivery_status_update_own
    ON messages.delivery_status
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
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
CREATE POLICY message_media_service_all
    ON messages.message_media
    FOR ALL
    USING (TRUE);
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
CREATE POLICY polls_service_all
    ON messages.polls
    FOR ALL
    USING (TRUE);
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
CREATE POLICY poll_options_service_all
    ON messages.poll_options
    FOR ALL
    USING (TRUE);
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
CREATE POLICY poll_votes_select_own
    ON messages.poll_votes
    FOR SELECT
    USING (user_id = auth.current_user_id());
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
CREATE POLICY poll_votes_delete_own
    ON messages.poll_votes
    FOR DELETE
    USING (user_id = auth.current_user_id());
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
CREATE POLICY typing_indicators_insert_own
    ON messages.typing_indicators
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY typing_indicators_delete_own
    ON messages.typing_indicators
    FOR DELETE
    USING (user_id = auth.current_user_id());
CREATE POLICY drafts_all_own
    ON messages.drafts
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY bookmarks_all_own
    ON messages.bookmarks
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY message_reports_select_reporter
    ON messages.message_reports
    FOR SELECT
    USING (reporter_user_id = auth.current_user_id());
CREATE POLICY message_reports_insert
    ON messages.message_reports
    FOR INSERT
    WITH CHECK (reporter_user_id = auth.current_user_id());
CREATE POLICY message_reports_admin_all
    ON messages.message_reports
    FOR ALL
    USING (auth.is_admin());
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
CREATE POLICY calls_service_all
    ON messages.calls
    FOR ALL
    USING (TRUE);
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
CREATE POLICY call_participants_service_all
    ON messages.call_participants
    FOR ALL
    USING (TRUE);
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
CREATE POLICY search_index_service_all
    ON messages.search_index
    FOR ALL
    USING (TRUE);
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
-- database-dup/rls/notifications.rls.sql
ALTER TABLE notifications.notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.push_delivery_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.email_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.sms_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.user_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.conversation_channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.notification_actions ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.action_responses ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.announcements ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.announcement_views ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.user_stats ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications.subscriptions ENABLE ROW LEVEL SECURITY;
CREATE POLICY notifications_select_own
    ON notifications.notifications
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        AND deleted_at IS NULL
    );
CREATE POLICY notifications_insert_service
    ON notifications.notifications
    FOR INSERT
    WITH CHECK (TRUE);
CREATE POLICY notifications_update_own
    ON notifications.notifications
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (
        user_id = auth.current_user_id()
    );
CREATE POLICY notifications_delete_own
    ON notifications.notifications
    FOR DELETE
    USING (user_id = auth.current_user_id());
CREATE POLICY notifications_admin_all
    ON notifications.notifications
    FOR ALL
    USING (auth.is_admin());
CREATE POLICY push_delivery_select_own
    ON notifications.push_delivery_log
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY push_delivery_service_all
    ON notifications.push_delivery_log
    FOR ALL
    USING (TRUE);
CREATE POLICY email_notifications_select_own
    ON notifications.email_notifications
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY email_notifications_service_all
    ON notifications.email_notifications
    FOR ALL
    USING (TRUE);
CREATE POLICY sms_notifications_select_own
    ON notifications.sms_notifications
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY sms_notifications_service_all
    ON notifications.sms_notifications
    FOR ALL
    USING (TRUE);
CREATE POLICY user_preferences_all_own
    ON notifications.user_preferences
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY conversation_channels_all_own
    ON notifications.conversation_channels
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY templates_select_active
    ON notifications.templates
    FOR SELECT
    USING (is_active = TRUE);
CREATE POLICY templates_admin_all
    ON notifications.templates
    FOR ALL
    USING (auth.is_admin());
CREATE POLICY notification_actions_select
    ON notifications.notification_actions
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM notifications.notifications n
            WHERE n.id = notifications.notification_actions.notification_id
            AND n.user_id = auth.current_user_id()
        )
    );
CREATE POLICY notification_actions_service_all
    ON notifications.notification_actions
    FOR ALL
    USING (TRUE);
CREATE POLICY action_responses_select_own
    ON notifications.action_responses
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY action_responses_insert_own
    ON notifications.action_responses
    FOR INSERT
    WITH CHECK (
        user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM notifications.notifications n
            WHERE n.id = notifications.action_responses.notification_id
            AND n.user_id = auth.current_user_id()
        )
    );
CREATE POLICY batches_admin_all
    ON notifications.batches
    FOR ALL
    USING (auth.is_admin());
CREATE POLICY batches_select_creator
    ON notifications.batches
    FOR SELECT
    USING (created_by_user_id = auth.current_user_id());
CREATE POLICY announcements_select_active
    ON notifications.announcements
    FOR SELECT
    USING (
        is_active = TRUE
        AND (starts_at IS NULL OR starts_at <= NOW())
        AND (ends_at IS NULL OR ends_at > NOW())
        AND (
            target_audience = 'all'
            OR auth.current_user_id() = ANY(target_user_ids)
        )
    );
CREATE POLICY announcements_admin_all
    ON notifications.announcements
    FOR ALL
    USING (auth.is_admin());
CREATE POLICY announcement_views_select_own
    ON notifications.announcement_views
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY announcement_views_insert_own
    ON notifications.announcement_views
    FOR INSERT
    WITH CHECK (
        user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM notifications.announcements a
            WHERE a.id = notifications.announcement_views.announcement_id
            AND a.is_active = TRUE
            AND (a.starts_at IS NULL OR a.starts_at <= NOW())
            AND (a.ends_at IS NULL OR a.ends_at > NOW())
        )
    );
CREATE POLICY announcement_views_update_own
    ON notifications.announcement_views
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY user_stats_select_own
    ON notifications.user_stats
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY user_stats_service_all
    ON notifications.user_stats
    FOR ALL
    USING (TRUE);
CREATE POLICY user_stats_select_admin
    ON notifications.user_stats
    FOR SELECT
    USING (auth.is_admin());
CREATE POLICY subscriptions_all_own
    ON notifications.subscriptions
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
-- database-dup/rls/users.rls.sql
ALTER TABLE users.profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.contact_groups ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.blocked_users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.privacy_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.status_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.status_views ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.activity_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.achievements ENABLE ROW LEVEL SECURITY;
ALTER TABLE users.reports ENABLE ROW LEVEL SECURITY;
CREATE POLICY profiles_select_public
    ON users.profiles
    FOR SELECT
    USING (
        profile_visibility = 'public'
        AND search_visibility = TRUE
        AND deactivated_at IS NULL
    );
CREATE POLICY profiles_select_own
    ON users.profiles
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY profiles_select_friends
    ON users.profiles
    FOR SELECT
    USING (
        profile_visibility = 'friends'
        AND deactivated_at IS NULL
        AND EXISTS (
            SELECT 1 FROM users.contacts
            WHERE contact_user_id = users.profiles.user_id
            AND user_id = auth.current_user_id()
            AND status = 'accepted'
        )
    );
CREATE POLICY profiles_update_own
    ON users.profiles
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY profiles_select_admin
    ON users.profiles
    FOR SELECT
    USING (auth.is_admin());
CREATE POLICY contacts_select_own
    ON users.contacts
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY contacts_insert_own
    ON users.contacts
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY contacts_update_own
    ON users.contacts
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY contacts_delete_own
    ON users.contacts
    FOR DELETE
    USING (user_id = auth.current_user_id());
CREATE POLICY contacts_select_incoming
    ON users.contacts
    FOR SELECT
    USING (contact_user_id = auth.current_user_id());
CREATE POLICY contact_groups_all_own
    ON users.contact_groups
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY settings_all_own
    ON users.settings
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY blocked_users_select_own
    ON users.blocked_users
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY blocked_users_insert_own
    ON users.blocked_users
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY blocked_users_update_own
    ON users.blocked_users
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY blocked_users_delete_own
    ON users.blocked_users
    FOR DELETE
    USING (user_id = auth.current_user_id());
CREATE POLICY privacy_overrides_all_own
    ON users.privacy_overrides
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY status_history_select_public
    ON users.status_history
    FOR SELECT
    USING (
        privacy = 'public'
        AND expires_at > NOW()
        AND deleted_at IS NULL
    );
CREATE POLICY status_history_select_own
    ON users.status_history
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY status_history_select_contacts
    ON users.status_history
    FOR SELECT
    USING (
        privacy = 'contacts'
        AND expires_at > NOW()
        AND deleted_at IS NULL
        AND EXISTS (
            SELECT 1 FROM users.contacts
            WHERE contact_user_id = users.status_history.user_id
            AND user_id = auth.current_user_id()
            AND status = 'accepted'
        )
    );
CREATE POLICY status_history_insert_own
    ON users.status_history
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY status_history_update_own
    ON users.status_history
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY status_history_delete_own
    ON users.status_history
    FOR DELETE
    USING (user_id = auth.current_user_id());
CREATE POLICY status_views_select_own
    ON users.status_views
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM users.status_history
            WHERE id = users.status_views.status_id
            AND user_id = auth.current_user_id()
        )
    );
CREATE POLICY status_views_insert
    ON users.status_views
    FOR INSERT
    WITH CHECK (viewer_user_id = auth.current_user_id());
CREATE POLICY activity_log_select_own
    ON users.activity_log
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY activity_log_insert_service
    ON users.activity_log
    FOR INSERT
    WITH CHECK (TRUE);
CREATE POLICY activity_log_select_admin
    ON users.activity_log
    FOR SELECT
    USING (auth.is_admin());
CREATE POLICY preferences_all_own
    ON users.preferences
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY devices_select_own
    ON users.devices
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY devices_insert_own
    ON users.devices
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY devices_update_own
    ON users.devices
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());
CREATE POLICY devices_delete_own
    ON users.devices
    FOR DELETE
    USING (user_id = auth.current_user_id());
CREATE POLICY achievements_select_own
    ON users.achievements
    FOR SELECT
    USING (user_id = auth.current_user_id());
CREATE POLICY achievements_select_public
    ON users.achievements
    FOR SELECT
    USING (
        display_on_profile = TRUE
        AND is_unlocked = TRUE
    );
CREATE POLICY achievements_insert_service
    ON users.achievements
    FOR INSERT
    WITH CHECK (TRUE);
CREATE POLICY achievements_update_display
    ON users.achievements
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (
        user_id = auth.current_user_id()
    );
CREATE POLICY reports_select_reporter
    ON users.reports
    FOR SELECT
    USING (reporter_user_id = auth.current_user_id());
CREATE POLICY reports_insert
    ON users.reports
    FOR INSERT
    WITH CHECK (reporter_user_id = auth.current_user_id());
CREATE POLICY reports_admin_all
    ON users.reports
    FOR ALL
    USING (auth.is_admin());
CREATE POLICY reports_select_assigned
    ON users.reports
    FOR SELECT
    USING (assigned_to = auth.current_user_id());
-- database-dup/schemas/analytics-schema.sql
CREATE SCHEMA IF NOT EXISTS analytics;
CREATE TABLE analytics.events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    event_name VARCHAR(255) NOT NULL, 
    event_category VARCHAR(100), 
    event_action VARCHAR(100),
    event_label VARCHAR(255),
    event_value DECIMAL(10,2),
    screen_name VARCHAR(255),
    previous_screen VARCHAR(255),
    flow_name VARCHAR(100), 
    device_id VARCHAR(255),
    platform VARCHAR(50), 
    os_name VARCHAR(100),
    os_version VARCHAR(50),
    app_version VARCHAR(50),
    app_build VARCHAR(50),
    ip_address INET,
    country VARCHAR(100),
    region VARCHAR(100),
    city VARCHAR(100),
    timezone VARCHAR(100),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    connection_type VARCHAR(50), 
    carrier VARCHAR(100),
    event_duration_ms INTEGER,
    load_time_ms INTEGER,
    ttfb_ms INTEGER, 
    properties JSONB DEFAULT '{}'::JSONB,
    utm_source VARCHAR(255),
    utm_medium VARCHAR(255),
    utm_campaign VARCHAR(255),
    utm_term VARCHAR(255),
    utm_content VARCHAR(255),
    referrer TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    event_timestamp TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE CASCADE,
    session_start TIMESTAMPTZ NOT NULL,
    session_end TIMESTAMPTZ,
    session_duration_seconds INTEGER,
    page_views INTEGER DEFAULT 0,
    screen_views INTEGER DEFAULT 0,
    event_count INTEGER DEFAULT 0,
    messages_sent INTEGER DEFAULT 0,
    messages_received INTEGER DEFAULT 0,
    device_id VARCHAR(255),
    device_type VARCHAR(50),
    platform VARCHAR(50),
    app_version VARCHAR(50),
    country VARCHAR(100),
    city VARCHAR(100),
    ip_address INET,
    is_engaged BOOLEAN DEFAULT FALSE, 
    bounce BOOLEAN DEFAULT TRUE,
    traffic_source VARCHAR(100),
    campaign VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.daily_active_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    sessions_count INTEGER DEFAULT 1,
    messages_sent INTEGER DEFAULT 0,
    messages_received INTEGER DEFAULT 0,
    time_spent_seconds INTEGER DEFAULT 0,
    features_used TEXT[],
    platforms_used VARCHAR(50)[], 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, user_id)
);
CREATE TABLE analytics.user_cohorts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    cohort_date DATE NOT NULL, 
    cohort_week INTEGER, 
    cohort_month INTEGER, 
    day_1_active BOOLEAN DEFAULT FALSE,
    day_7_active BOOLEAN DEFAULT FALSE,
    day_14_active BOOLEAN DEFAULT FALSE,
    day_30_active BOOLEAN DEFAULT FALSE,
    day_60_active BOOLEAN DEFAULT FALSE,
    day_90_active BOOLEAN DEFAULT FALSE,
    messages_sent_total INTEGER DEFAULT 0,
    days_active_count INTEGER DEFAULT 0,
    last_active_date DATE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);
CREATE TABLE analytics.funnels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    funnel_name VARCHAR(100) NOT NULL, 
    step_name VARCHAR(100) NOT NULL,
    step_order INTEGER NOT NULL,
    entered_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    time_in_step_seconds INTEGER,
    is_completed BOOLEAN DEFAULT FALSE,
    dropped_off BOOLEAN DEFAULT FALSE,
    drop_off_reason TEXT,
    properties JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.feature_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    feature_name VARCHAR(100) NOT NULL, 
    feature_category VARCHAR(100),
    usage_count INTEGER DEFAULT 1,
    first_used_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ DEFAULT NOW(),
    total_duration_seconds INTEGER DEFAULT 0,
    engagement_score DECIMAL(10,2), 
    date DATE DEFAULT CURRENT_DATE,
    UNIQUE(user_id, feature_name, date)
);
CREATE TABLE analytics.ab_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_name VARCHAR(255) UNIQUE NOT NULL,
    test_description TEXT,
    variants JSONB NOT NULL, 
    target_percentage INTEGER DEFAULT 100, 
    target_platforms VARCHAR(50)[],
    target_countries VARCHAR(5)[],
    target_user_segments TEXT[],
    status VARCHAR(50) DEFAULT 'draft', 
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    sample_size INTEGER DEFAULT 0,
    confidence_level DECIMAL(5,2),
    winning_variant VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by_user_id UUID REFERENCES auth.users(id)
);
CREATE TABLE analytics.ab_test_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_id UUID NOT NULL REFERENCES analytics.ab_tests(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    variant_name VARCHAR(100) NOT NULL,
    variant_config JSONB,
    converted BOOLEAN DEFAULT FALSE,
    conversion_value DECIMAL(10,2),
    converted_at TIMESTAMPTZ,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(test_id, user_id)
);
CREATE TABLE analytics.performance_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_type VARCHAR(100) NOT NULL, 
    metric_name VARCHAR(255) NOT NULL,
    metric_value DECIMAL(15,4) NOT NULL,
    metric_unit VARCHAR(50), 
    service_name VARCHAR(100), 
    endpoint VARCHAR(255),
    method VARCHAR(20), 
    status_code INTEGER,
    duration_ms INTEGER,
    memory_used_mb INTEGER,
    cpu_percentage DECIMAL(5,2),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    ip_address INET,
    is_error BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    error_stack TEXT,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.error_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    error_type VARCHAR(100) NOT NULL, 
    error_message TEXT NOT NULL,
    error_code VARCHAR(100),
    error_stack TEXT,
    severity VARCHAR(20) DEFAULT 'error', 
    service_name VARCHAR(100),
    function_name VARCHAR(255),
    file_path TEXT,
    line_number INTEGER,
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    device_id VARCHAR(255),
    http_method VARCHAR(20),
    endpoint VARCHAR(255),
    request_id VARCHAR(255),
    request_body JSONB,
    response_body JSONB,
    environment VARCHAR(50), 
    app_version VARCHAR(50),
    platform VARCHAR(50),
    occurrences INTEGER DEFAULT 1,
    first_occurred_at TIMESTAMPTZ DEFAULT NOW(),
    last_occurred_at TIMESTAMPTZ DEFAULT NOW(),
    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    resolved_by_user_id UUID REFERENCES auth.users(id),
    resolution_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE analytics.revenue_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    transaction_type VARCHAR(100) NOT NULL, 
    product_id VARCHAR(255),
    product_name VARCHAR(255),
    product_category VARCHAR(100),
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'USD',
    amount_usd DECIMAL(10,2), 
    payment_method VARCHAR(100), 
    payment_provider VARCHAR(100),
    transaction_id VARCHAR(255) UNIQUE,
    status VARCHAR(50) DEFAULT 'completed', 
    refunded_at TIMESTAMPTZ,
    refund_amount DECIMAL(10,2),
    refund_reason TEXT,
    campaign_source VARCHAR(255),
    is_subscription BOOLEAN DEFAULT FALSE,
    subscription_period VARCHAR(50), 
    is_trial BOOLEAN DEFAULT FALSE,
    is_renewal BOOLEAN DEFAULT FALSE,
    transaction_date TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.user_ltv (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    total_revenue DECIMAL(10,2) DEFAULT 0.00,
    total_transactions INTEGER DEFAULT 0,
    average_transaction_value DECIMAL(10,2) DEFAULT 0.00,
    days_active INTEGER DEFAULT 0,
    messages_sent_total INTEGER DEFAULT 0,
    messages_received_total INTEGER DEFAULT 0,
    predicted_ltv_30d DECIMAL(10,2),
    predicted_ltv_90d DECIMAL(10,2),
    predicted_ltv_365d DECIMAL(10,2),
    user_segment VARCHAR(100), 
    churn_risk_score DECIMAL(5,2), 
    last_calculated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.daily_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL UNIQUE,
    dau INTEGER DEFAULT 0, 
    new_users INTEGER DEFAULT 0,
    churned_users INTEGER DEFAULT 0,
    total_messages_sent BIGINT DEFAULT 0,
    total_sessions BIGINT DEFAULT 0,
    avg_session_duration_seconds INTEGER DEFAULT 0,
    avg_messages_per_user DECIMAL(10,2) DEFAULT 0.00,
    images_uploaded INTEGER DEFAULT 0,
    videos_uploaded INTEGER DEFAULT 0,
    voice_messages_sent INTEGER DEFAULT 0,
    new_conversations INTEGER DEFAULT 0,
    new_groups_created INTEGER DEFAULT 0,
    new_friendships INTEGER DEFAULT 0,
    voice_calls_total INTEGER DEFAULT 0,
    video_calls_total INTEGER DEFAULT 0,
    total_call_duration_minutes BIGINT DEFAULT 0,
    revenue_total DECIMAL(10,2) DEFAULT 0.00,
    new_subscriptions INTEGER DEFAULT 0,
    cancelled_subscriptions INTEGER DEFAULT 0,
    avg_api_latency_ms INTEGER,
    error_count INTEGER DEFAULT 0,
    uptime_percentage DECIMAL(5,2) DEFAULT 100.00,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.user_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    segment_name VARCHAR(100) NOT NULL, 
    segment_category VARCHAR(100),
    criteria_met JSONB,
    confidence_score DECIMAL(5,2), 
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    UNIQUE(user_id, segment_name)
);
CREATE TABLE analytics.page_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    page_url TEXT,
    page_title VARCHAR(255),
    screen_name VARCHAR(255),
    screen_class VARCHAR(255),
    referrer_url TEXT,
    referrer_source VARCHAR(255),
    view_duration_seconds INTEGER,
    time_to_interactive_ms INTEGER,
    scroll_depth_percentage INTEGER,
    clicks_count INTEGER DEFAULT 0,
    device_id VARCHAR(255),
    platform VARCHAR(50),
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.search_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    search_query TEXT NOT NULL,
    search_type VARCHAR(50), 
    search_category VARCHAR(100),
    results_count INTEGER DEFAULT 0,
    results_clicked INTEGER DEFAULT 0,
    clicked_position INTEGER, 
    search_duration_ms INTEGER,
    screen_name VARCHAR(255),
    searched_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.content_engagement (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    content_type VARCHAR(100) NOT NULL, 
    content_id UUID NOT NULL,
    viewed BOOLEAN DEFAULT FALSE,
    viewed_at TIMESTAMPTZ,
    view_duration_ms INTEGER,
    liked BOOLEAN DEFAULT FALSE,
    liked_at TIMESTAMPTZ,
    shared BOOLEAN DEFAULT FALSE,
    shared_at TIMESTAMPTZ,
    share_count INTEGER DEFAULT 0,
    saved BOOLEAN DEFAULT FALSE,
    saved_at TIMESTAMPTZ,
    commented BOOLEAN DEFAULT FALSE,
    comment_count INTEGER DEFAULT 0,
    completion_percentage INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, content_type, content_id)
);
CREATE TABLE analytics.push_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    notification_type VARCHAR(100),
    platform VARCHAR(50),
    sent_count INTEGER DEFAULT 0,
    delivered_count INTEGER DEFAULT 0,
    opened_count INTEGER DEFAULT 0,
    dismissed_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    delivery_rate DECIMAL(5,2) DEFAULT 0.00,
    open_rate DECIMAL(5,2) DEFAULT 0.00,
    avg_delivery_time_seconds INTEGER,
    avg_time_to_open_seconds INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, notification_type, platform)
);
CREATE TABLE analytics.api_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(20) NOT NULL,
    service_name VARCHAR(100),
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    api_key_id UUID REFERENCES auth.api_keys(id) ON DELETE SET NULL,
    request_size_bytes INTEGER,
    request_headers JSONB,
    query_params JSONB,
    status_code INTEGER NOT NULL,
    response_size_bytes INTEGER,
    response_time_ms INTEGER,
    db_query_time_ms INTEGER,
    cache_hit BOOLEAN DEFAULT FALSE,
    ip_address INET,
    country VARCHAR(100),
    is_error BOOLEAN DEFAULT FALSE,
    error_code VARCHAR(100),
    error_message TEXT,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_url TEXT NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', 
    http_status_code INTEGER,
    response_time_ms INTEGER,
    attempt_number INTEGER DEFAULT 1,
    max_attempts INTEGER DEFAULT 3,
    next_retry_at TIMESTAMPTZ,
    payload_size_bytes INTEGER,
    error_message TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.user_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    feedback_type VARCHAR(100) NOT NULL, 
    feedback_category VARCHAR(100),
    rating INTEGER, 
    title VARCHAR(255),
    description TEXT,
    screen_name VARCHAR(255),
    app_version VARCHAR(50),
    platform VARCHAR(50),
    screenshot_urls TEXT[],
    log_data JSONB,
    status VARCHAR(50) DEFAULT 'new', 
    priority VARCHAR(20) DEFAULT 'medium',
    response_text TEXT,
    responded_at TIMESTAMPTZ,
    responded_by_user_id UUID REFERENCES auth.users(id),
    sentiment_score DECIMAL(5,2), 
    sentiment_label VARCHAR(50), 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    referred_user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    referral_code VARCHAR(100) UNIQUE NOT NULL,
    referral_link TEXT,
    referral_method VARCHAR(50), 
    status VARCHAR(50) DEFAULT 'pending', 
    clicked_at TIMESTAMPTZ,
    registered_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    reward_given BOOLEAN DEFAULT FALSE,
    reward_type VARCHAR(100),
    reward_value DECIMAL(10,2),
    reward_given_at TIMESTAMPTZ,
    utm_source VARCHAR(255),
    utm_campaign VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.churn_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    churn_probability DECIMAL(5,2) NOT NULL, 
    churn_risk_level VARCHAR(20), 
    contributing_factors JSONB, 
    predicted_churn_date DATE,
    days_until_churn INTEGER,
    intervention_recommended VARCHAR(255),
    intervention_sent BOOLEAN DEFAULT FALSE,
    intervention_sent_at TIMESTAMPTZ,
    actually_churned BOOLEAN,
    churn_date DATE,
    prediction_date DATE DEFAULT CURRENT_DATE,
    model_version VARCHAR(50),
    confidence_score DECIMAL(5,2),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.attribution (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    conversion_type VARCHAR(100) NOT NULL, 
    conversion_value DECIMAL(10,2),
    converted_at TIMESTAMPTZ NOT NULL,
    attribution_model VARCHAR(50) DEFAULT 'last_click', 
    first_touch_source VARCHAR(255),
    first_touch_medium VARCHAR(255),
    first_touch_campaign VARCHAR(255),
    first_touch_at TIMESTAMPTZ,
    last_touch_source VARCHAR(255),
    last_touch_medium VARCHAR(255),
    last_touch_campaign VARCHAR(255),
    last_touch_at TIMESTAMPTZ,
    touchpoint_count INTEGER DEFAULT 0,
    journey_duration_hours INTEGER,
    touchpoints JSONB DEFAULT '[]'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE analytics.heatmap_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    screen_name VARCHAR(255) NOT NULL,
    element_id VARCHAR(255),
    element_type VARCHAR(100), 
    interaction_type VARCHAR(50) NOT NULL, 
    x_coordinate INTEGER,
    y_coordinate INTEGER,
    viewport_width INTEGER,
    viewport_height INTEGER,
    interaction_count INTEGER DEFAULT 1,
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    platform VARCHAR(50),
    device_type VARCHAR(50),
    date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
-- database-dup/schemas/auth-schema.sql
CREATE SCHEMA IF NOT EXISTS auth;
CREATE TABLE auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone_number VARCHAR(20) UNIQUE,
    phone_country_code VARCHAR(5),
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    password_hash TEXT NOT NULL,
    password_salt TEXT NOT NULL,
    password_algorithm VARCHAR(50) DEFAULT 'bcrypt',
    password_last_changed_at TIMESTAMPTZ,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret TEXT,
    two_factor_backup_codes TEXT[],
    account_status VARCHAR(50) DEFAULT 'active', 
    account_locked_until TIMESTAMPTZ,
    failed_login_attempts INTEGER DEFAULT 0,
    last_failed_login_at TIMESTAMPTZ,
    last_successful_login_at TIMESTAMPTZ,
    requires_password_change BOOLEAN DEFAULT FALSE,
    password_history JSONB DEFAULT '[]'::JSONB, 
    security_questions JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    created_by_ip INET,
    created_by_user_agent TEXT
);
CREATE TABLE auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    session_token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    device_id VARCHAR(255),
    device_name VARCHAR(255),
    device_type VARCHAR(50), 
    device_os VARCHAR(100),
    device_os_version VARCHAR(50),
    device_model VARCHAR(100),
    device_manufacturer VARCHAR(100),
    browser_name VARCHAR(100),
    browser_version VARCHAR(50),
    user_agent TEXT,
    ip_address INET NOT NULL,
    ip_country VARCHAR(100),
    ip_region VARCHAR(100),
    ip_city VARCHAR(100),
    ip_timezone VARCHAR(100),
    ip_isp VARCHAR(255),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    is_mobile BOOLEAN DEFAULT FALSE,
    is_trusted_device BOOLEAN DEFAULT FALSE,
    fcm_token TEXT, 
    apns_token TEXT, 
    push_enabled BOOLEAN DEFAULT TRUE,
    session_type VARCHAR(50) DEFAULT 'user', 
    expires_at TIMESTAMPTZ NOT NULL,
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    last_refresh_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE auth.otp_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    identifier VARCHAR(255) NOT NULL, 
    identifier_type VARCHAR(20) NOT NULL, 
    otp_code VARCHAR(10) NOT NULL,
    otp_hash TEXT NOT NULL,
    purpose VARCHAR(50) NOT NULL, 
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 5,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    sent_via VARCHAR(50), 
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE auth.oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, 
    provider_user_id VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    provider_username VARCHAR(255),
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    scope TEXT[],
    profile_data JSONB,
    is_primary BOOLEAN DEFAULT FALSE,
    linked_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    unlinked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);
CREATE TABLE auth.password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    email_sent_at TIMESTAMPTZ,
    email_opened_at TIMESTAMPTZ,
    link_clicked_at TIMESTAMPTZ
);
CREATE TABLE auth.email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token TEXT UNIQUE NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    attempts INTEGER DEFAULT 0
);
CREATE TABLE auth.security_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    event_type VARCHAR(100) NOT NULL, 
    event_category VARCHAR(50), 
    severity VARCHAR(20) DEFAULT 'info', 
    status VARCHAR(20), 
    description TEXT,
    ip_address INET,
    user_agent TEXT,
    device_id VARCHAR(255),
    location_country VARCHAR(100),
    location_city VARCHAR(100),
    risk_score INTEGER, 
    is_suspicious BOOLEAN DEFAULT FALSE,
    blocked_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE auth.login_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    login_method VARCHAR(50), 
    status VARCHAR(20), 
    failure_reason TEXT,
    ip_address INET,
    user_agent TEXT,
    device_id VARCHAR(255),
    device_fingerprint TEXT,
    location_country VARCHAR(100),
    location_city VARCHAR(100),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    is_new_device BOOLEAN DEFAULT FALSE,
    is_new_location BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE auth.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE, 
    key_prefix VARCHAR(20) NOT NULL, 
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    service_name VARCHAR(100), 
    scopes TEXT[], 
    rate_limit_per_hour INTEGER DEFAULT 1000,
    is_active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
COMMENT ON TABLE auth.users IS 'Core authentication table for user accounts';
COMMENT ON TABLE auth.sessions IS 'Active user sessions with device and location tracking';
COMMENT ON TABLE auth.otp_verifications IS 'One-time password verification for 2FA and account verification';
COMMENT ON TABLE auth.oauth_providers IS 'OAuth social login integrations';
COMMENT ON TABLE auth.security_events IS 'Audit log for security-related events';
COMMENT ON TABLE auth.login_history IS 'Historical record of all login attempts';
COMMENT ON TABLE auth.api_keys IS 'API keys for programmatic access';
-- database-dup/schemas/location-ip-schema.sql
CREATE SCHEMA IF NOT EXISTS location;
CREATE TABLE location.ip_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_address INET UNIQUE NOT NULL,
    ip_version INTEGER, 
    country_code VARCHAR(5),
    country_name VARCHAR(100),
    region_code VARCHAR(10),
    region_name VARCHAR(100),
    city VARCHAR(100),
    postal_code VARCHAR(20),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    metro_code VARCHAR(20),
    timezone VARCHAR(100),
    isp VARCHAR(255),
    organization VARCHAR(255),
    asn VARCHAR(50), 
    as_organization VARCHAR(255),
    connection_type VARCHAR(50), 
    user_type VARCHAR(50), 
    is_proxy BOOLEAN DEFAULT FALSE,
    is_vpn BOOLEAN DEFAULT FALSE,
    is_tor BOOLEAN DEFAULT FALSE,
    is_hosting BOOLEAN DEFAULT FALSE,
    is_anonymous BOOLEAN DEFAULT FALSE,
    threat_level VARCHAR(20), 
    risk_score INTEGER, 
    is_bogon BOOLEAN DEFAULT FALSE, 
    first_seen_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    lookup_count INTEGER DEFAULT 1,
    user_count INTEGER DEFAULT 0, 
    lookup_provider VARCHAR(50), 
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE location.user_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    ip_address_id UUID REFERENCES location.ip_addresses(id),
    ip_address INET NOT NULL,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    accuracy_meters INTEGER,
    altitude DECIMAL(10, 2),
    country VARCHAR(100),
    region VARCHAR(100),
    city VARCHAR(100),
    postal_code VARCHAR(20),
    address TEXT,
    location_type VARCHAR(50), 
    location_source VARCHAR(50), 
    is_new_location BOOLEAN DEFAULT FALSE,
    is_new_country BOOLEAN DEFAULT FALSE,
    is_new_city BOOLEAN DEFAULT FALSE,
    distance_from_previous_km DECIMAL(10, 2),
    timezone VARCHAR(100),
    timezone_offset INTEGER, 
    device_id VARCHAR(255),
    captured_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE location.location_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    shared_with_user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    shared_with_conversation_id UUID REFERENCES messages.conversations(id) ON DELETE CASCADE,
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    accuracy_meters INTEGER,
    altitude DECIMAL(10, 2),
    heading DECIMAL(5, 2), 
    speed_mps DECIMAL(10, 2), 
    share_type VARCHAR(50) DEFAULT 'temporary', 
    duration_minutes INTEGER, 
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    is_live BOOLEAN DEFAULT TRUE, 
    update_interval_seconds INTEGER DEFAULT 30,
    show_exact_location BOOLEAN DEFAULT TRUE,
    show_address BOOLEAN DEFAULT FALSE,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    stopped_at TIMESTAMPTZ,
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE location.location_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_share_id UUID NOT NULL REFERENCES location.location_shares(id) ON DELETE CASCADE,
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    accuracy_meters INTEGER,
    altitude DECIMAL(10, 2),
    heading DECIMAL(5, 2),
    speed_mps DECIMAL(10, 2),
    battery_level INTEGER, 
    is_charging BOOLEAN,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE location.places (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    place_name VARCHAR(255) NOT NULL,
    place_type VARCHAR(100), 
    place_category VARCHAR(100),
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    address TEXT,
    city VARCHAR(100),
    region VARCHAR(100),
    country VARCHAR(100),
    postal_code VARCHAR(20),
    phone_number VARCHAR(20),
    website_url TEXT,
    google_place_id VARCHAR(255) UNIQUE,
    foursquare_id VARCHAR(255),
    rating DECIMAL(3, 2),
    review_count INTEGER DEFAULT 0,
    price_level INTEGER, 
    check_in_count INTEGER DEFAULT 0,
    share_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE location.check_ins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    place_id UUID REFERENCES location.places(id) ON DELETE SET NULL,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    place_name VARCHAR(255),
    caption TEXT,
    photo_url TEXT,
    visibility VARCHAR(50) DEFAULT 'friends', 
    like_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    checked_in_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE location.geofences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    fence_name VARCHAR(255) NOT NULL,
    fence_type VARCHAR(50), 
    center_latitude DECIMAL(10, 8),
    center_longitude DECIMAL(11, 8),
    radius_meters INTEGER,
    polygon_coordinates JSONB, 
    trigger_on_enter BOOLEAN DEFAULT TRUE,
    trigger_on_exit BOOLEAN DEFAULT FALSE,
    trigger_on_dwell BOOLEAN DEFAULT FALSE,
    dwell_time_seconds INTEGER,
    action_type VARCHAR(100), 
    action_config JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE location.geofence_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    geofence_id UUID NOT NULL REFERENCES location.geofences(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, 
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    action_executed BOOLEAN DEFAULT FALSE,
    action_result JSONB,
    occurred_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE location.nearby_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    nearby_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    distance_meters INTEGER NOT NULL,
    location_name VARCHAR(255), 
    has_interacted BOOLEAN DEFAULT FALSE,
    interaction_type VARCHAR(50), 
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    last_nearby_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, nearby_user_id, detected_at)
);
CREATE TABLE location.location_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    recommendation_type VARCHAR(100), 
    recommended_user_id UUID REFERENCES auth.users(id),
    recommended_place_id UUID REFERENCES location.places(id),
    relevance_score DECIMAL(5, 2), 
    distance_meters INTEGER,
    is_shown BOOLEAN DEFAULT FALSE,
    shown_at TIMESTAMPTZ,
    is_acted_upon BOOLEAN DEFAULT FALSE,
    action_type VARCHAR(50),
    acted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);
CREATE TABLE location.region_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_type VARCHAR(50) NOT NULL, 
    region_code VARCHAR(10),
    region_name VARCHAR(255) NOT NULL,
    total_users INTEGER DEFAULT 0,
    active_users_daily INTEGER DEFAULT 0,
    active_users_monthly INTEGER DEFAULT 0,
    new_users_today INTEGER DEFAULT 0,
    messages_sent_today BIGINT DEFAULT 0,
    calls_made_today INTEGER DEFAULT 0,
    growth_rate_percentage DECIMAL(5, 2),
    date DATE DEFAULT CURRENT_DATE,
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(region_type, region_code, date)
);
CREATE TABLE location.ip_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_address INET NOT NULL,
    ip_range CIDR, 
    blacklist_reason VARCHAR(255) NOT NULL,
    blacklist_type VARCHAR(50), 
    severity VARCHAR(20) DEFAULT 'medium',
    incident_count INTEGER DEFAULT 1,
    evidence_urls TEXT[],
    notes TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    blacklisted_by_user_id UUID REFERENCES auth.users(id),
    source VARCHAR(100), 
    blacklisted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE location.vpn_detection_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    ip_address INET NOT NULL,
    is_vpn BOOLEAN DEFAULT FALSE,
    is_proxy BOOLEAN DEFAULT FALSE,
    is_tor BOOLEAN DEFAULT FALSE,
    is_hosting BOOLEAN DEFAULT FALSE,
    vpn_provider VARCHAR(255),
    proxy_type VARCHAR(50),
    confidence_score DECIMAL(5, 2), 
    action_taken VARCHAR(100), 
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
-- database-dup/schemas/media-schema.sql
CREATE SCHEMA IF NOT EXISTS media;
CREATE TABLE media.files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploader_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE SET NULL,
    file_name VARCHAR(500) NOT NULL,
    original_file_name VARCHAR(500),
    file_type VARCHAR(100) NOT NULL, 
    mime_type VARCHAR(255) NOT NULL,
    file_category VARCHAR(50), 
    file_extension VARCHAR(20),
    file_size_bytes BIGINT NOT NULL,
    storage_provider VARCHAR(50) DEFAULT 'r2', 
    storage_bucket VARCHAR(255),
    storage_key TEXT NOT NULL, 
    storage_url TEXT NOT NULL, 
    storage_region VARCHAR(100),
    cdn_url TEXT, 
    has_thumbnail BOOLEAN DEFAULT FALSE,
    thumbnail_url TEXT,
    thumbnail_small_url TEXT, 
    thumbnail_medium_url TEXT, 
    thumbnail_large_url TEXT, 
    has_preview BOOLEAN DEFAULT FALSE,
    preview_url TEXT,
    width INTEGER,
    height INTEGER,
    duration_seconds INTEGER, 
    bitrate INTEGER,
    frame_rate DECIMAL(10,2),
    codec VARCHAR(100),
    resolution VARCHAR(50), 
    aspect_ratio VARCHAR(20),
    color_profile VARCHAR(100),
    orientation INTEGER, 
    has_alpha_channel BOOLEAN,
    dominant_colors TEXT[], 
    video_codec VARCHAR(100),
    audio_codec VARCHAR(100),
    subtitle_tracks JSONB DEFAULT '[]'::JSONB,
    audio_channels INTEGER,
    sample_rate INTEGER,
    page_count INTEGER,
    word_count INTEGER,
    processing_status VARCHAR(50) DEFAULT 'pending', 
    processing_started_at TIMESTAMPTZ,
    processing_completed_at TIMESTAMPTZ,
    processing_error TEXT,
    processing_attempts INTEGER DEFAULT 0,
    is_encrypted BOOLEAN DEFAULT FALSE,
    encryption_key_id TEXT,
    content_hash VARCHAR(255), 
    checksum VARCHAR(255), 
    is_scanned BOOLEAN DEFAULT FALSE,
    virus_scan_status VARCHAR(50), 
    virus_scan_at TIMESTAMPTZ,
    moderation_status VARCHAR(50) DEFAULT 'pending', 
    moderation_score DECIMAL(5,2), 
    moderation_labels JSONB DEFAULT '[]'::JSONB, 
    is_nsfw BOOLEAN DEFAULT FALSE,
    nsfw_score DECIMAL(5,2),
    moderated_at TIMESTAMPTZ,
    moderated_by_user_id UUID REFERENCES auth.users(id),
    visibility VARCHAR(50) DEFAULT 'private', 
    access_token TEXT UNIQUE,
    expires_at TIMESTAMPTZ,
    max_downloads INTEGER,
    download_count INTEGER DEFAULT 0,
    view_count INTEGER DEFAULT 0,
    is_compressed BOOLEAN DEFAULT FALSE,
    compression_ratio DECIMAL(5,2),
    original_file_size_bytes BIGINT,
    exif_data JSONB DEFAULT '{}'::JSONB,
    gps_latitude DECIMAL(10, 8),
    gps_longitude DECIMAL(11, 8),
    gps_altitude DECIMAL(10, 2),
    camera_make VARCHAR(255),
    camera_model VARCHAR(255),
    lens_model VARCHAR(255),
    focal_length DECIMAL(10,2),
    aperture DECIMAL(10,2),
    iso INTEGER,
    shutter_speed VARCHAR(50),
    capture_date TIMESTAMPTZ,
    last_accessed_at TIMESTAMPTZ,
    access_count BIGINT DEFAULT 0,
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    permanently_delete_at TIMESTAMPTZ, 
    uploaded_from_device_id VARCHAR(255),
    uploaded_from_ip INET,
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE media.processing_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    task_type VARCHAR(100) NOT NULL, 
    priority INTEGER DEFAULT 5, 
    status VARCHAR(50) DEFAULT 'queued', 
    attempt_count INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    worker_id VARCHAR(255),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    input_params JSONB DEFAULT '{}'::JSONB,
    output_result JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE media.thumbnails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    size_type VARCHAR(50) NOT NULL, 
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    file_size_bytes BIGINT,
    storage_key TEXT NOT NULL,
    storage_url TEXT NOT NULL,
    format VARCHAR(20), 
    quality INTEGER, 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(file_id, size_type)
);
CREATE TABLE media.transcoding_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    output_file_id UUID REFERENCES media.files(id),
    profile_name VARCHAR(100) NOT NULL, 
    status VARCHAR(50) DEFAULT 'pending',
    progress_percentage INTEGER DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    estimated_completion_at TIMESTAMPTZ,
    error_message TEXT,
    transcoding_params JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE media.albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    cover_file_id UUID REFERENCES media.files(id),
    album_type VARCHAR(50) DEFAULT 'custom', 
    is_system_album BOOLEAN DEFAULT FALSE,
    file_count INTEGER DEFAULT 0,
    visibility VARCHAR(50) DEFAULT 'private',
    sort_order VARCHAR(50) DEFAULT 'date_desc', 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE media.album_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    album_id UUID NOT NULL REFERENCES media.albums(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    display_order INTEGER,
    added_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(album_id, file_id)
);
CREATE TABLE media.tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    tag_name VARCHAR(100) NOT NULL,
    tag_type VARCHAR(50) DEFAULT 'user', 
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, tag_name, tag_type)
);
CREATE TABLE media.file_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES media.tags(id) ON DELETE CASCADE,
    confidence_score DECIMAL(5,2), 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(file_id, tag_id)
);
CREATE TABLE media.shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    shared_by_user_id UUID NOT NULL REFERENCES auth.users(id),
    shared_with_user_id UUID REFERENCES auth.users(id),
    shared_with_conversation_id UUID REFERENCES messages.conversations(id),
    share_token TEXT UNIQUE,
    access_type VARCHAR(50) DEFAULT 'view', 
    password_hash TEXT,
    expires_at TIMESTAMPTZ,
    max_views INTEGER,
    view_count INTEGER DEFAULT 0,
    download_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);
CREATE TABLE media.access_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    access_type VARCHAR(50) NOT NULL, 
    ip_address INET,
    user_agent TEXT,
    device_id VARCHAR(255),
    referrer TEXT,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    bytes_transferred BIGINT,
    access_duration_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE media.sticker_packs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_user_id UUID REFERENCES auth.users(id),
    pack_name VARCHAR(255) NOT NULL,
    pack_description TEXT,
    cover_file_id UUID REFERENCES media.files(id),
    icon_file_id UUID REFERENCES media.files(id),
    sticker_count INTEGER DEFAULT 0,
    is_official BOOLEAN DEFAULT FALSE,
    is_animated BOOLEAN DEFAULT FALSE,
    is_public BOOLEAN DEFAULT FALSE,
    download_count INTEGER DEFAULT 0,
    install_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE media.stickers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_user_id UUID REFERENCES auth.users(id),
    sticker_pack_id UUID REFERENCES media.sticker_packs(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    sticker_name VARCHAR(255),
    emojis TEXT[], 
    is_animated BOOLEAN DEFAULT FALSE,
    usage_count BIGINT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE media.user_sticker_packs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    sticker_pack_id UUID NOT NULL REFERENCES media.sticker_packs(id) ON DELETE CASCADE,
    display_order INTEGER,
    installed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, sticker_pack_id)
);
CREATE TABLE media.gifs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50), 
    provider_gif_id VARCHAR(255),
    title TEXT,
    url TEXT NOT NULL,
    preview_url TEXT,
    thumbnail_url TEXT,
    width INTEGER,
    height INTEGER,
    file_size_bytes BIGINT,
    tags TEXT[],
    usage_count BIGINT DEFAULT 0,
    is_trending BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_gif_id)
);
CREATE TABLE media.favorite_gifs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    gif_id UUID NOT NULL REFERENCES media.gifs(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, gif_id)
);
CREATE TABLE media.storage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    total_files INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    images_count INTEGER DEFAULT 0,
    images_size_bytes BIGINT DEFAULT 0,
    videos_count INTEGER DEFAULT 0,
    videos_size_bytes BIGINT DEFAULT 0,
    audio_count INTEGER DEFAULT 0,
    audio_size_bytes BIGINT DEFAULT 0,
    documents_count INTEGER DEFAULT 0,
    documents_size_bytes BIGINT DEFAULT 0,
    storage_quota_bytes BIGINT DEFAULT 5368709120, 
    storage_used_percentage DECIMAL(5,2) DEFAULT 0.00,
    last_calculated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
-- database-dup/schemas/message-schema.sql
CREATE SCHEMA IF NOT EXISTS messages;
CREATE TABLE messages.conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_type VARCHAR(50) NOT NULL, 
    title VARCHAR(255),
    description TEXT,
    avatar_url TEXT,
    creator_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE SET NULL,
    is_group BOOLEAN DEFAULT FALSE,
    is_channel BOOLEAN DEFAULT FALSE,
    is_encrypted BOOLEAN DEFAULT TRUE,
    encryption_key_id TEXT,
    max_members INTEGER,
    is_public BOOLEAN DEFAULT FALSE,
    invite_link TEXT UNIQUE,
    invite_link_expires_at TIMESTAMPTZ,
    join_approval_required BOOLEAN DEFAULT FALSE,
    who_can_send_messages VARCHAR(50) DEFAULT 'all', 
    who_can_add_members VARCHAR(50) DEFAULT 'admins',
    who_can_edit_info VARCHAR(50) DEFAULT 'admins',
    who_can_pin_messages VARCHAR(50) DEFAULT 'admins',
    is_active BOOLEAN DEFAULT TRUE,
    is_archived BOOLEAN DEFAULT FALSE,
    archived_at TIMESTAMPTZ,
    member_count INTEGER DEFAULT 0,
    message_count BIGINT DEFAULT 0,
    last_message_id UUID,
    last_message_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE messages.conversation_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'member', 
    nickname VARCHAR(100),
    custom_notifications BOOLEAN,
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    is_pinned BOOLEAN DEFAULT FALSE,
    pin_order INTEGER,
    is_archived BOOLEAN DEFAULT FALSE,
    last_read_message_id UUID,
    last_read_at TIMESTAMPTZ,
    unread_count INTEGER DEFAULT 0,
    mention_count INTEGER DEFAULT 0,
    can_send_messages BOOLEAN DEFAULT TRUE,
    can_send_media BOOLEAN DEFAULT TRUE,
    can_add_members BOOLEAN DEFAULT FALSE,
    can_remove_members BOOLEAN DEFAULT FALSE,
    can_edit_info BOOLEAN DEFAULT FALSE,
    can_pin_messages BOOLEAN DEFAULT FALSE,
    can_delete_messages BOOLEAN DEFAULT FALSE,
    join_method VARCHAR(50), 
    invited_by_user_id UUID REFERENCES auth.users(id),
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ,
    removed_by_user_id UUID REFERENCES auth.users(id),
    removal_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(conversation_id, user_id)
);
CREATE TABLE messages.messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    sender_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE SET NULL,
    parent_message_id UUID REFERENCES messages.messages(id) ON DELETE SET NULL, 
    message_type VARCHAR(50) DEFAULT 'text', 
    content TEXT,
    content_encrypted BOOLEAN DEFAULT TRUE,
    content_hash TEXT, 
    format_type VARCHAR(50) DEFAULT 'plain', 
    mentions JSONB DEFAULT '[]'::JSONB, 
    hashtags TEXT[],
    links JSONB DEFAULT '[]'::JSONB, 
    status VARCHAR(50) DEFAULT 'sent', 
    is_edited BOOLEAN DEFAULT FALSE,
    edited_at TIMESTAMPTZ,
    edit_history JSONB DEFAULT '[]'::JSONB,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_for VARCHAR(50), 
    is_pinned BOOLEAN DEFAULT FALSE,
    pinned_at TIMESTAMPTZ,
    pinned_by_user_id UUID REFERENCES auth.users(id),
    delivered_at TIMESTAMPTZ,
    delivery_count INTEGER DEFAULT 0,
    read_count INTEGER DEFAULT 0,
    is_flagged BOOLEAN DEFAULT FALSE,
    flag_reason TEXT,
    flagged_at TIMESTAMPTZ,
    flagged_by_user_id UUID REFERENCES auth.users(id),
    scheduled_at TIMESTAMPTZ,
    is_scheduled BOOLEAN DEFAULT FALSE,
    reply_count INTEGER DEFAULT 0,
    last_reply_at TIMESTAMPTZ,
    reaction_count INTEGER DEFAULT 0,
    is_forwarded BOOLEAN DEFAULT FALSE,
    forwarded_from_message_id UUID REFERENCES messages.messages(id),
    forward_count INTEGER DEFAULT 0,
    sent_from_device_id VARCHAR(255),
    sent_from_ip INET,
    expires_at TIMESTAMPTZ,
    expire_after_seconds INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE messages.reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    reaction_type VARCHAR(100) NOT NULL, 
    reaction_emoji VARCHAR(100),
    reaction_skin_tone VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id, reaction_type)
);
CREATE TABLE messages.delivery_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'sent', 
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    failed_reason TEXT,
    retry_count INTEGER DEFAULT 0,
    device_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id)
);
CREATE TABLE messages.message_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    media_id UUID NOT NULL, 
    media_type VARCHAR(50) NOT NULL, 
    display_order INTEGER DEFAULT 0,
    caption TEXT,
    thumbnail_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE messages.link_previews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    title TEXT,
    description TEXT,
    image_url TEXT,
    favicon_url TEXT,
    site_name VARCHAR(255),
    content_type VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, url)
);
CREATE TABLE messages.polls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID UNIQUE NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    allow_multiple_answers BOOLEAN DEFAULT FALSE,
    is_anonymous BOOLEAN DEFAULT FALSE,
    is_quiz BOOLEAN DEFAULT FALSE,
    correct_option_id INTEGER,
    explanation TEXT,
    closes_at TIMESTAMPTZ,
    is_closed BOOLEAN DEFAULT FALSE,
    closed_at TIMESTAMPTZ,
    total_votes INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE messages.poll_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES messages.polls(id) ON DELETE CASCADE,
    option_text TEXT NOT NULL,
    option_order INTEGER NOT NULL,
    vote_count INTEGER DEFAULT 0,
    vote_percentage DECIMAL(5,2) DEFAULT 0.00,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE messages.poll_votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES messages.polls(id) ON DELETE CASCADE,
    poll_option_id UUID NOT NULL REFERENCES messages.poll_options(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    voted_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(poll_id, poll_option_id, user_id)
);
CREATE TABLE messages.typing_indicators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL, 
    UNIQUE(conversation_id, user_id)
);
CREATE TABLE messages.message_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    reporter_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    report_type VARCHAR(100) NOT NULL, 
    report_category VARCHAR(100),
    description TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    priority VARCHAR(20) DEFAULT 'medium',
    assigned_to UUID REFERENCES auth.users(id),
    resolution TEXT,
    action_taken VARCHAR(100), 
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE messages.drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    content TEXT,
    reply_to_message_id UUID REFERENCES messages.messages(id),
    mentions JSONB DEFAULT '[]'::JSONB,
    attachments JSONB DEFAULT '[]'::JSONB, 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, conversation_id)
);
CREATE TABLE messages.bookmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    collection_name VARCHAR(255), 
    notes TEXT,
    tags TEXT[],
    bookmarked_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, message_id)
);
CREATE TABLE messages.pinned_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    pinned_by_user_id UUID NOT NULL REFERENCES auth.users(id),
    pin_order INTEGER DEFAULT 0,
    pinned_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(conversation_id, message_id)
);
CREATE TABLE messages.conversation_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    inviter_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    invitee_user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    invitee_phone_number VARCHAR(20),
    invitee_email VARCHAR(255),
    invite_code TEXT UNIQUE,
    status VARCHAR(50) DEFAULT 'pending', 
    max_uses INTEGER DEFAULT 1,
    use_count INTEGER DEFAULT 0,
    expires_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE messages.search_index (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID UNIQUE NOT NULL REFERENCES messages.messages(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id),
    content_tsvector TSVECTOR,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE messages.calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    call_type VARCHAR(50) NOT NULL, 
    initiator_user_id UUID NOT NULL REFERENCES auth.users(id),
    status VARCHAR(50) DEFAULT 'initiated', 
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    video_quality VARCHAR(50),
    audio_quality VARCHAR(50),
    connection_quality VARCHAR(50),
    packet_loss_percentage DECIMAL(5,2),
    media_server_id VARCHAR(255),
    room_id VARCHAR(255),
    end_reason VARCHAR(100), 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE messages.call_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id UUID NOT NULL REFERENCES messages.calls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'invited', 
    joined_at TIMESTAMPTZ,
    left_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    is_video_enabled BOOLEAN DEFAULT FALSE,
    is_audio_enabled BOOLEAN DEFAULT TRUE,
    is_screen_sharing BOOLEAN DEFAULT FALSE,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(call_id, user_id)
);
CREATE TABLE messages.conversation_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID UNIQUE NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    disappearing_messages_enabled BOOLEAN DEFAULT FALSE,
    disappearing_messages_duration INTEGER, 
    message_history_enabled BOOLEAN DEFAULT TRUE,
    screenshot_notification BOOLEAN DEFAULT FALSE,
    read_receipts_enabled BOOLEAN DEFAULT TRUE,
    typing_indicators_enabled BOOLEAN DEFAULT TRUE,
    link_previews_enabled BOOLEAN DEFAULT TRUE,
    auto_download_media BOOLEAN DEFAULT TRUE,
    message_request_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
-- database-dup/schemas/notification-schema.sql
CREATE SCHEMA IF NOT EXISTS notifications;
CREATE TABLE notifications.notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    notification_type VARCHAR(100) NOT NULL, 
    notification_category VARCHAR(50), 
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    summary TEXT, 
    icon_url TEXT,
    image_url TEXT,
    sound VARCHAR(100) DEFAULT 'default',
    badge_count INTEGER,
    related_user_id UUID REFERENCES auth.users(id),
    related_message_id UUID REFERENCES messages.messages(id),
    related_conversation_id UUID REFERENCES messages.conversations(id),
    related_call_id UUID REFERENCES messages.calls(id),
    action_url TEXT,
    action_type VARCHAR(100), 
    action_data JSONB DEFAULT '{}'::JSONB,
    is_read BOOLEAN DEFAULT FALSE,
    is_seen BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    seen_at TIMESTAMPTZ,
    delivery_status VARCHAR(50) DEFAULT 'pending', 
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_reason TEXT,
    retry_count INTEGER DEFAULT 0,
    priority VARCHAR(20) DEFAULT 'normal', 
    scheduled_for TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    group_key VARCHAR(255), 
    group_count INTEGER DEFAULT 1,
    is_group_summary BOOLEAN DEFAULT FALSE,
    device_id VARCHAR(255),
    platform VARCHAR(50), 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE notifications.push_delivery_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications.notifications(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    device_id VARCHAR(255),
    push_token TEXT NOT NULL,
    push_provider VARCHAR(50) NOT NULL, 
    status VARCHAR(50) DEFAULT 'pending', 
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    dismissed_at TIMESTAMPTZ,
    provider_message_id VARCHAR(255),
    provider_response JSONB,
    error_code VARCHAR(100),
    error_message TEXT,
    time_to_deliver_ms INTEGER,
    time_to_open_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE notifications.email_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    notification_id UUID REFERENCES notifications.notifications(id) ON DELETE SET NULL,
    email_to VARCHAR(255) NOT NULL,
    email_from VARCHAR(255) DEFAULT 'notifications@messaging.app',
    reply_to VARCHAR(255),
    subject VARCHAR(500) NOT NULL,
    body_text TEXT NOT NULL,
    body_html TEXT,
    template_name VARCHAR(100),
    template_data JSONB DEFAULT '{}'::JSONB,
    status VARCHAR(50) DEFAULT 'pending', 
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    clicked_at TIMESTAMPTZ,
    bounced_at TIMESTAMPTZ,
    bounce_reason TEXT,
    provider VARCHAR(50) DEFAULT 'gcp',
    provider_message_id VARCHAR(255),
    provider_response JSONB,
    open_count INTEGER DEFAULT 0,
    click_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE notifications.sms_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    notification_id UUID REFERENCES notifications.notifications(id) ON DELETE SET NULL,
    phone_number VARCHAR(20) NOT NULL,
    country_code VARCHAR(5),
    message TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', 
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    failure_reason TEXT,
    provider VARCHAR(50) DEFAULT 'gcp',
    provider_message_id VARCHAR(255),
    provider_response JSONB,
    segment_count INTEGER DEFAULT 1,
    cost_per_sms DECIMAL(10,4),
    total_cost DECIMAL(10,4),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE notifications.user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    push_enabled BOOLEAN DEFAULT TRUE,
    email_enabled BOOLEAN DEFAULT TRUE,
    sms_enabled BOOLEAN DEFAULT FALSE,
    in_app_enabled BOOLEAN DEFAULT TRUE,
    message_push BOOLEAN DEFAULT TRUE,
    message_email BOOLEAN DEFAULT FALSE,
    message_sms BOOLEAN DEFAULT FALSE,
    mention_push BOOLEAN DEFAULT TRUE,
    mention_email BOOLEAN DEFAULT TRUE,
    mention_sms BOOLEAN DEFAULT FALSE,
    reaction_push BOOLEAN DEFAULT TRUE,
    reaction_email BOOLEAN DEFAULT FALSE,
    reaction_sms BOOLEAN DEFAULT FALSE,
    call_push BOOLEAN DEFAULT TRUE,
    call_email BOOLEAN DEFAULT FALSE,
    call_sms BOOLEAN DEFAULT TRUE,
    missed_call_push BOOLEAN DEFAULT TRUE,
    friend_request_push BOOLEAN DEFAULT TRUE,
    friend_request_email BOOLEAN DEFAULT TRUE,
    friend_accept_push BOOLEAN DEFAULT TRUE,
    group_invite_push BOOLEAN DEFAULT TRUE,
    group_invite_email BOOLEAN DEFAULT TRUE,
    group_message_push BOOLEAN DEFAULT TRUE,
    group_mention_push BOOLEAN DEFAULT TRUE,
    security_alerts_push BOOLEAN DEFAULT TRUE,
    security_alerts_email BOOLEAN DEFAULT TRUE,
    security_alerts_sms BOOLEAN DEFAULT TRUE,
    account_updates_email BOOLEAN DEFAULT TRUE,
    marketing_push BOOLEAN DEFAULT FALSE,
    marketing_email BOOLEAN DEFAULT FALSE,
    promotional_email BOOLEAN DEFAULT FALSE,
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    quiet_hours_timezone VARCHAR(100),
    quiet_hours_days INTEGER[], 
    bundle_notifications BOOLEAN DEFAULT TRUE,
    bundle_interval_minutes INTEGER DEFAULT 5,
    notification_sound VARCHAR(100) DEFAULT 'default',
    vibration_enabled BOOLEAN DEFAULT TRUE,
    led_notification BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE notifications.conversation_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES messages.conversations(id) ON DELETE CASCADE,
    notifications_enabled BOOLEAN DEFAULT TRUE,
    push_enabled BOOLEAN,
    email_enabled BOOLEAN,
    sound_enabled BOOLEAN,
    vibration_enabled BOOLEAN,
    custom_sound VARCHAR(100),
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    mute_reason VARCHAR(100), 
    priority_level VARCHAR(20) DEFAULT 'normal', 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, conversation_id)
);
CREATE TABLE notifications.templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name VARCHAR(100) UNIQUE NOT NULL,
    template_type VARCHAR(50) NOT NULL, 
    title_template TEXT,
    body_template TEXT NOT NULL,
    html_template TEXT, 
    required_variables TEXT[],
    optional_variables TEXT[],
    language_code VARCHAR(10) DEFAULT 'en',
    is_active BOOLEAN DEFAULT TRUE,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by_user_id UUID REFERENCES auth.users(id)
);
CREATE TABLE notifications.notification_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications.notifications(id) ON DELETE CASCADE,
    action_id VARCHAR(100) NOT NULL, 
    action_label VARCHAR(100) NOT NULL,
    action_type VARCHAR(50), 
    action_url TEXT,
    action_data JSONB DEFAULT '{}'::JSONB,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE notifications.action_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications.notifications(id) ON DELETE CASCADE,
    action_id VARCHAR(100) NOT NULL,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    response_type VARCHAR(50), 
    response_data JSONB DEFAULT '{}'::JSONB,
    device_id VARCHAR(255),
    responded_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE notifications.batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_name VARCHAR(255),
    batch_type VARCHAR(50), 
    target_user_ids UUID[],
    target_segment VARCHAR(100), 
    target_count INTEGER,
    notification_data JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', 
    priority VARCHAR(20) DEFAULT 'normal',
    sent_count INTEGER DEFAULT 0,
    delivered_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    opened_count INTEGER DEFAULT 0,
    scheduled_for TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    rate_limit_per_second INTEGER DEFAULT 100,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by_user_id UUID REFERENCES auth.users(id),
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE notifications.announcements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    announcement_type VARCHAR(50), 
    severity VARCHAR(20) DEFAULT 'info', 
    icon_url TEXT,
    image_url TEXT,
    background_color VARCHAR(7),
    action_label VARCHAR(100),
    action_url TEXT,
    action_type VARCHAR(50),
    target_audience VARCHAR(100) DEFAULT 'all', 
    target_user_ids UUID[],
    min_app_version VARCHAR(50),
    max_app_version VARCHAR(50),
    target_countries VARCHAR(5)[],
    display_frequency VARCHAR(50) DEFAULT 'once', 
    max_display_count INTEGER DEFAULT 1,
    display_priority INTEGER DEFAULT 5,
    is_active BOOLEAN DEFAULT TRUE,
    is_dismissible BOOLEAN DEFAULT TRUE,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    view_count INTEGER DEFAULT 0,
    click_count INTEGER DEFAULT 0,
    dismiss_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by_user_id UUID REFERENCES auth.users(id)
);
CREATE TABLE notifications.announcement_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    announcement_id UUID NOT NULL REFERENCES notifications.announcements(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    view_count INTEGER DEFAULT 1,
    first_viewed_at TIMESTAMPTZ DEFAULT NOW(),
    last_viewed_at TIMESTAMPTZ DEFAULT NOW(),
    clicked BOOLEAN DEFAULT FALSE,
    clicked_at TIMESTAMPTZ,
    dismissed BOOLEAN DEFAULT FALSE,
    dismissed_at TIMESTAMPTZ,
    UNIQUE(announcement_id, user_id)
);
CREATE TABLE notifications.user_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    total_notifications_sent INTEGER DEFAULT 0,
    total_notifications_delivered INTEGER DEFAULT 0,
    total_notifications_opened INTEGER DEFAULT 0,
    total_notifications_dismissed INTEGER DEFAULT 0,
    push_sent INTEGER DEFAULT 0,
    push_delivered INTEGER DEFAULT 0,
    push_opened INTEGER DEFAULT 0,
    email_sent INTEGER DEFAULT 0,
    email_delivered INTEGER DEFAULT 0,
    email_opened INTEGER DEFAULT 0,
    email_clicked INTEGER DEFAULT 0,
    sms_sent INTEGER DEFAULT 0,
    sms_delivered INTEGER DEFAULT 0,
    last_notification_at TIMESTAMPTZ,
    last_opened_notification_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE notifications.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    topic_name VARCHAR(255) NOT NULL, 
    is_subscribed BOOLEAN DEFAULT TRUE,
    subscribed_at TIMESTAMPTZ DEFAULT NOW(),
    unsubscribed_at TIMESTAMPTZ,
    UNIQUE(user_id, topic_name)
);
-- database-dup/schemas/user-schema.sql
CREATE SCHEMA IF NOT EXISTS users;
CREATE TABLE users.profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    username VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    middle_name VARCHAR(100),
    bio TEXT,
    bio_links JSONB DEFAULT '[]'::JSONB, 
    avatar_url TEXT,
    avatar_thumbnail_url TEXT,
    cover_image_url TEXT,
    date_of_birth DATE,
    gender VARCHAR(50),
    pronouns VARCHAR(50),
    language_code VARCHAR(10) DEFAULT 'en',
    timezone VARCHAR(100),
    country_code VARCHAR(5),
    city VARCHAR(100),
    phone_visible BOOLEAN DEFAULT FALSE,
    email_visible BOOLEAN DEFAULT FALSE,
    online_status VARCHAR(20) DEFAULT 'offline', 
    last_seen_at TIMESTAMPTZ,
    profile_visibility VARCHAR(20) DEFAULT 'public', 
    search_visibility BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    website_url TEXT,
    social_links JSONB DEFAULT '{}'::JSONB, 
    interests TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deactivated_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::JSONB
);
CREATE TABLE users.contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    contact_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) DEFAULT 'contact', 
    status VARCHAR(50) DEFAULT 'pending', 
    nickname VARCHAR(100),
    notes TEXT,
    is_favorite BOOLEAN DEFAULT FALSE,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    custom_notifications JSONB,
    contact_source VARCHAR(50), 
    contact_groups TEXT[],
    last_interaction_at TIMESTAMPTZ,
    interaction_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,
    blocked_at TIMESTAMPTZ,
    block_reason TEXT,
    UNIQUE(user_id, contact_user_id),
    CHECK (user_id != contact_user_id)
);
CREATE TABLE users.contact_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    group_name VARCHAR(100) NOT NULL,
    group_color VARCHAR(7), 
    group_icon VARCHAR(50),
    description TEXT,
    member_count INTEGER DEFAULT 0,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, group_name)
);
CREATE TABLE users.settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    profile_visibility VARCHAR(20) DEFAULT 'public',
    last_seen_visibility VARCHAR(20) DEFAULT 'everyone', 
    online_status_visibility VARCHAR(20) DEFAULT 'everyone',
    profile_photo_visibility VARCHAR(20) DEFAULT 'everyone',
    about_visibility VARCHAR(20) DEFAULT 'everyone',
    read_receipts_enabled BOOLEAN DEFAULT TRUE,
    typing_indicators_enabled BOOLEAN DEFAULT TRUE,
    push_notifications_enabled BOOLEAN DEFAULT TRUE,
    email_notifications_enabled BOOLEAN DEFAULT TRUE,
    sms_notifications_enabled BOOLEAN DEFAULT FALSE,
    message_notifications BOOLEAN DEFAULT TRUE,
    group_message_notifications BOOLEAN DEFAULT TRUE,
    mention_notifications BOOLEAN DEFAULT TRUE,
    reaction_notifications BOOLEAN DEFAULT TRUE,
    call_notifications BOOLEAN DEFAULT TRUE,
    notification_sound VARCHAR(100) DEFAULT 'default',
    vibration_enabled BOOLEAN DEFAULT TRUE,
    notification_preview VARCHAR(20) DEFAULT 'full', 
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    enter_key_to_send BOOLEAN DEFAULT FALSE,
    auto_download_photos BOOLEAN DEFAULT TRUE,
    auto_download_videos BOOLEAN DEFAULT FALSE,
    auto_download_documents BOOLEAN DEFAULT FALSE,
    auto_download_on_wifi_only BOOLEAN DEFAULT TRUE,
    compress_images BOOLEAN DEFAULT TRUE,
    save_to_gallery BOOLEAN DEFAULT FALSE,
    chat_backup_enabled BOOLEAN DEFAULT TRUE,
    chat_backup_frequency VARCHAR(20) DEFAULT 'daily',
    screen_lock_enabled BOOLEAN DEFAULT FALSE,
    screen_lock_timeout INTEGER DEFAULT 0, 
    fingerprint_unlock BOOLEAN DEFAULT FALSE,
    face_unlock BOOLEAN DEFAULT FALSE,
    show_security_notifications BOOLEAN DEFAULT TRUE,
    theme VARCHAR(20) DEFAULT 'system', 
    font_size VARCHAR(20) DEFAULT 'medium',
    chat_wallpaper TEXT,
    use_system_emoji BOOLEAN DEFAULT TRUE,
    language_code VARCHAR(10) DEFAULT 'en',
    timezone VARCHAR(100),
    date_format VARCHAR(20) DEFAULT 'MM/DD/YYYY',
    time_format VARCHAR(20) DEFAULT '12h',
    low_data_mode BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE users.blocked_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    blocked_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    block_reason TEXT,
    blocked_at TIMESTAMPTZ DEFAULT NOW(),
    unblocked_at TIMESTAMPTZ,
    block_type VARCHAR(50) DEFAULT 'full', 
    metadata JSONB DEFAULT '{}'::JSONB,
    UNIQUE(user_id, blocked_user_id),
    CHECK (user_id != blocked_user_id)
);
CREATE TABLE users.privacy_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    target_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    last_seen_visible BOOLEAN,
    online_status_visible BOOLEAN,
    profile_photo_visible BOOLEAN,
    about_visible BOOLEAN,
    read_receipts_enabled BOOLEAN,
    typing_indicators_enabled BOOLEAN,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, target_user_id)
);
CREATE TABLE users.status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    status_text TEXT,
    status_emoji VARCHAR(100),
    media_url TEXT,
    media_type VARCHAR(50),
    background_color VARCHAR(7),
    views_count INTEGER DEFAULT 0,
    privacy VARCHAR(20) DEFAULT 'public', 
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TABLE users.status_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status_id UUID NOT NULL REFERENCES users.status_history(id) ON DELETE CASCADE,
    viewer_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    view_duration INTEGER, 
    UNIQUE(status_id, viewer_user_id)
);
CREATE TABLE users.activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    activity_type VARCHAR(100) NOT NULL, 
    activity_category VARCHAR(50),
    description TEXT,
    old_value JSONB,
    new_value JSONB,
    ip_address INET,
    user_agent TEXT,
    device_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE users.preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    preference_key VARCHAR(255) NOT NULL,
    preference_value JSONB NOT NULL,
    category VARCHAR(100),
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, preference_key)
);
CREATE TABLE users.devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    device_id VARCHAR(255) NOT NULL,
    device_name VARCHAR(255),
    device_type VARCHAR(50),
    device_model VARCHAR(100),
    device_manufacturer VARCHAR(100),
    os_name VARCHAR(100),
    os_version VARCHAR(50),
    app_version VARCHAR(50),
    is_current_device BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    last_active_at TIMESTAMPTZ DEFAULT NOW(),
    registered_at TIMESTAMPTZ DEFAULT NOW(),
    fcm_token TEXT,
    apns_token TEXT,
    push_enabled BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'::JSONB,
    UNIQUE(user_id, device_id)
);
CREATE TABLE users.achievements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    achievement_type VARCHAR(100) NOT NULL, 
    achievement_name VARCHAR(255),
    achievement_description TEXT,
    achievement_icon TEXT,
    achievement_rarity VARCHAR(50), 
    progress INTEGER DEFAULT 0,
    progress_total INTEGER,
    is_unlocked BOOLEAN DEFAULT FALSE,
    unlocked_at TIMESTAMPTZ,
    display_on_profile BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE users.reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    reported_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    report_type VARCHAR(100) NOT NULL, 
    report_category VARCHAR(100),
    description TEXT,
    evidence_urls TEXT[],
    status VARCHAR(50) DEFAULT 'pending', 
    priority VARCHAR(20) DEFAULT 'medium',
    assigned_to UUID REFERENCES auth.users(id),
    resolution TEXT,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
COMMENT ON TABLE users.profiles IS 'Public user profile information';
COMMENT ON TABLE users.contacts IS 'User relationships and contacts';
COMMENT ON TABLE users.settings IS 'User application settings and preferences';
COMMENT ON TABLE users.blocked_users IS 'Users that have been blocked';
COMMENT ON TABLE users.status_history IS 'Temporary status updates (stories)';
COMMENT ON TABLE users.devices IS 'User registered devices';
-- database-dup/triggers/auth/01_timestamp.triggers.sql
CREATE TRIGGER trigger_auth_users_updated_at
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();
CREATE TRIGGER trigger_auth_oauth_updated_at
    BEFORE UPDATE ON auth.oauth_providers
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();
CREATE TRIGGER trigger_auth_api_keys_updated_at
    BEFORE UPDATE ON auth.api_keys
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

-- database-dup/triggers/auth/02_security_logging.triggers.sql
CREATE TRIGGER trigger_auth_users_password_change
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    WHEN (OLD.password_hash IS DISTINCT FROM NEW.password_hash)
    EXECUTE FUNCTION auth.log_password_change();
CREATE TRIGGER trigger_auth_users_2fa_change
    AFTER UPDATE ON auth.users
    FOR EACH ROW
    WHEN (OLD.two_factor_enabled IS DISTINCT FROM NEW.two_factor_enabled)
    EXECUTE FUNCTION auth.log_2fa_change();
CREATE TRIGGER trigger_auth_login_history_log_failed_attempts
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    WHEN (NEW.status = 'failure')
    EXECUTE FUNCTION auth.log_failed_login_attempt();

-- database-dup/triggers/auth/03_session_lifecycle.triggers.sql
CREATE TRIGGER trigger_auth_users_failed_login_attempts
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_failed_login_attempts();
CREATE TRIGGER trigger_auth_sessions_successful_login
    AFTER INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_last_successful_login();
CREATE TRIGGER trigger_auth_login_history_device_fingerprint
    BEFORE INSERT ON auth.login_history
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_device_fingerprint();
CREATE TRIGGER trigger_auth_sessions_log_creation
    AFTER INSERT ON auth.sessions
    FOR EACH ROW
    EXECUTE FUNCTION auth.log_session_creation();
CREATE TRIGGER trigger_auth_sessions_log_revocation
    AFTER UPDATE ON auth.sessions
    FOR EACH ROW
    WHEN (NEW.revoked_at IS NOT NULL AND OLD.revoked_at IS NULL)
    EXECUTE FUNCTION auth.log_session_revocation();

-- database-dup/triggers/auth/04_account_protection.triggers.sql
CREATE TRIGGER trigger_auth_users_prevent_deletion
    BEFORE DELETE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.prevent_user_deletion_with_active_sessions();
CREATE TRIGGER trigger_auth_users_cleanup_oauth
    BEFORE DELETE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.cleanup_user_oauth_providers();

-- database-dup/triggers/media/01_timestamp.triggers.sql
CREATE TRIGGER trigger_media_files_updated_at
    BEFORE UPDATE ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();
CREATE TRIGGER trigger_media_albums_updated_at
    BEFORE UPDATE ON media.albums
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();
CREATE TRIGGER trigger_media_sticker_packs_updated_at
    BEFORE UPDATE ON media.sticker_packs
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();
CREATE TRIGGER trigger_media_processing_queue_updated_at
    BEFORE UPDATE ON media.processing_queue
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();
CREATE TRIGGER trigger_media_transcoding_jobs_updated_at
    BEFORE UPDATE ON media.transcoding_jobs
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();
CREATE TRIGGER trigger_media_storage_stats_updated_at
    BEFORE UPDATE ON media.storage_stats
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- database-dup/triggers/media/02_storage_management.triggers.sql
CREATE TRIGGER trigger_media_files_validate_quota
    BEFORE INSERT ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.validate_storage_quota();
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
CREATE TRIGGER trigger_auth_users_create_storage_stats
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION media.create_default_storage_stats();
CREATE TRIGGER trigger_media_files_set_deletion_date
    BEFORE UPDATE ON media.files
    FOR EACH ROW
    WHEN (NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL)
    EXECUTE FUNCTION media.set_permanent_deletion_date();

-- database-dup/triggers/media/03_processing_pipeline.triggers.sql
CREATE TRIGGER trigger_media_files_queue_thumbnail
    AFTER INSERT ON media.files
    FOR EACH ROW
    WHEN (NEW.file_category IN ('image', 'video'))
    EXECUTE FUNCTION media.queue_thumbnail_generation();
CREATE TRIGGER trigger_media_files_queue_scan
    AFTER INSERT ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.queue_virus_scan();

-- database-dup/triggers/media/04_count_tracking.triggers.sql
CREATE TRIGGER trigger_media_access_log_increment
    AFTER INSERT ON media.access_log
    FOR EACH ROW
    EXECUTE FUNCTION media.increment_access_count();
CREATE TRIGGER trigger_media_access_log_download
    AFTER INSERT ON media.access_log
    FOR EACH ROW
    WHEN (NEW.access_type IN ('download', 'view'))
    EXECUTE FUNCTION media.increment_download_count();
CREATE TRIGGER trigger_media_access_log_share_counts
    AFTER INSERT ON media.access_log
    FOR EACH ROW
    WHEN (NEW.access_type IN ('view', 'download'))
    EXECUTE FUNCTION media.increment_share_counts();
CREATE TRIGGER trigger_media_album_files_count_insert
    AFTER INSERT ON media.album_files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_album_file_count();
CREATE TRIGGER trigger_media_album_files_count_delete
    AFTER DELETE ON media.album_files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_album_file_count();
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
CREATE TRIGGER trigger_media_file_tags_usage_insert
    AFTER INSERT ON media.file_tags
    FOR EACH ROW
    EXECUTE FUNCTION media.update_tag_usage_count();
CREATE TRIGGER trigger_media_file_tags_usage_delete
    AFTER DELETE ON media.file_tags
    FOR EACH ROW
    EXECUTE FUNCTION media.update_tag_usage_count();

-- database-dup/triggers/messages/01_timestamp.triggers.sql
CREATE TRIGGER trigger_messages_conversations_updated_at
    BEFORE UPDATE ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();
CREATE TRIGGER trigger_messages_participants_updated_at
    BEFORE UPDATE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();
CREATE TRIGGER trigger_messages_updated_at
    BEFORE UPDATE ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();
CREATE TRIGGER trigger_messages_reports_updated_at
    BEFORE UPDATE ON messages.message_reports
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();
CREATE TRIGGER trigger_messages_drafts_updated_at
    BEFORE UPDATE ON messages.drafts
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- database-dup/triggers/messages/02_conversation_lifecycle.triggers.sql
CREATE TRIGGER trigger_conversations_updated_at
    BEFORE UPDATE ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();
CREATE TRIGGER trigger_conversations_create_settings
    AFTER INSERT ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.create_default_conversation_settings();
CREATE TRIGGER trigger_conversations_add_creator
    AFTER INSERT ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.add_creator_as_participant();
CREATE TRIGGER trigger_participants_updated_at
    BEFORE UPDATE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();
CREATE TRIGGER trigger_participants_set_permissions
    BEFORE INSERT OR UPDATE OF role ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.set_participant_permissions();
CREATE TRIGGER trigger_participants_member_count_insert
    AFTER INSERT ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_member_count();
CREATE TRIGGER trigger_participants_member_count_update
    AFTER UPDATE ON messages.conversation_participants
    FOR EACH ROW
    WHEN (
        OLD.left_at     IS DISTINCT FROM NEW.left_at
        OR
        OLD.removed_at  IS DISTINCT FROM NEW.removed_at
    )
    EXECUTE FUNCTION messages.update_member_count();
CREATE TRIGGER trigger_participants_member_count_delete
    AFTER DELETE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_member_count();
-- database-dup/triggers/messages/03_message_lifecycle.triggers.sql
CREATE TRIGGER trigger_messages_update_conversation
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_conversation_last_message();
CREATE TRIGGER trigger_messages_set_expiration
    BEFORE INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.set_message_expiration();
CREATE TRIGGER trigger_messages_set_edited
    BEFORE UPDATE ON messages.messages
    FOR EACH ROW
    WHEN (OLD.content IS DISTINCT FROM NEW.content)
    EXECUTE FUNCTION messages.set_edited_timestamp();
CREATE TRIGGER trigger_messages_update_reply_count
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.parent_message_id IS NOT NULL)
    EXECUTE FUNCTION messages.update_reply_count();
CREATE TRIGGER trigger_messages_increment_forward_count
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.forwarded_from_message_id IS NOT NULL)
    EXECUTE FUNCTION messages.increment_forward_count();

-- database-dup/triggers/messages/04_delivery_tracking.triggers.sql
CREATE TRIGGER trigger_messages_create_delivery_status
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.create_delivery_status();
CREATE TRIGGER trigger_delivery_status_update_counts
    AFTER INSERT OR UPDATE ON messages.delivery_status
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_message_delivery_counts();
CREATE TRIGGER trigger_messages_increment_unread
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.increment_unread_count();
CREATE TRIGGER trigger_messages_increment_mentions
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.mentions IS NOT NULL AND jsonb_array_length(NEW.mentions) > 0)
    EXECUTE FUNCTION messages.increment_mention_count();

-- database-dup/triggers/messages/05_search_index.triggers.sql
CREATE TRIGGER trigger_messages_update_search_index
    AFTER INSERT OR UPDATE ON messages.messages
    FOR EACH ROW
    WHEN (NEW.message_type = 'text' AND NEW.content IS NOT NULL AND NOT NEW.is_deleted)
    EXECUTE FUNCTION messages.update_search_index();

-- database-dup/triggers/messages/06_reactions.triggers.sql
CREATE TRIGGER trigger_reactions_update_count_insert
    AFTER INSERT ON messages.reactions
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_reaction_count();
CREATE TRIGGER trigger_reactions_update_count_delete
    AFTER DELETE ON messages.reactions
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_reaction_count();

-- database-dup/triggers/messages/07_polls.triggers.sql
CREATE TRIGGER trigger_poll_votes_update_counts
    AFTER INSERT ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_poll_votes();
CREATE TRIGGER trigger_poll_votes_update_counts_delete
    AFTER DELETE ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_poll_votes();
CREATE TRIGGER trigger_polls_auto_close
    BEFORE UPDATE ON messages.polls
    FOR EACH ROW
    WHEN (NEW.closes_at IS NOT NULL AND NEW.closes_at <= NOW())
    EXECUTE FUNCTION messages.auto_close_poll();
CREATE TRIGGER trigger_poll_votes_validate_not_closed
    BEFORE INSERT ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_poll_not_closed();
CREATE TRIGGER trigger_poll_votes_validate_single
    BEFORE INSERT ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_single_vote();

-- database-dup/triggers/messages/08_validation.triggers.sql
CREATE TRIGGER trigger_messages_validate_sender
    BEFORE INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_participant_can_send();
CREATE TRIGGER trigger_messages_validate_not_blocked
    BEFORE INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_not_blocked();
CREATE TRIGGER trigger_typing_indicators_cleanup
    BEFORE INSERT ON messages.typing_indicators
    FOR EACH ROW
    EXECUTE FUNCTION messages.cleanup_typing_indicator();

-- database-dup/triggers/notifications/01_timestamp.triggers.sql
CREATE TRIGGER trigger_notifications_updated_at
    BEFORE UPDATE ON notifications.notifications
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();
CREATE TRIGGER trigger_user_preferences_updated_at
    BEFORE UPDATE ON notifications.user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();
CREATE TRIGGER trigger_conversation_channels_updated_at
    BEFORE UPDATE ON notifications.conversation_channels
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();
CREATE TRIGGER trigger_templates_updated_at
    BEFORE UPDATE ON notifications.templates
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();
CREATE TRIGGER trigger_announcements_updated_at
    BEFORE UPDATE ON notifications.announcements
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();
CREATE TRIGGER trigger_user_stats_updated_at
    BEFORE UPDATE ON notifications.user_stats
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_updated_at_column();

-- database-dup/triggers/notifications/02_user_setup.triggers.sql
CREATE TRIGGER trigger_auth_users_create_notification_preferences
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION notifications.create_default_preferences();
CREATE TRIGGER trigger_auth_users_create_notification_stats
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION notifications.create_default_notification_stats();

-- database-dup/triggers/notifications/03_stats_tracking.triggers.sql
CREATE TRIGGER trigger_notifications_update_stats_insert
    AFTER INSERT ON notifications.notifications
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_user_stats();
CREATE TRIGGER trigger_push_delivery_update_stats
    AFTER UPDATE ON notifications.push_delivery_log
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status OR OLD.opened_at IS DISTINCT FROM NEW.opened_at)
    EXECUTE FUNCTION notifications.update_delivery_stats();
CREATE TRIGGER trigger_email_notifications_update_stats
    AFTER UPDATE ON notifications.email_notifications
    FOR EACH ROW
    WHEN (
        OLD.status IS DISTINCT FROM NEW.status
        OR OLD.opened_at IS DISTINCT FROM NEW.opened_at
        OR OLD.clicked_at IS DISTINCT FROM NEW.clicked_at
    )
    EXECUTE FUNCTION notifications.update_email_stats();

-- database-dup/triggers/notifications/04_delivery_lifecycle.triggers.sql
CREATE TRIGGER trigger_push_delivery_update_notification
    AFTER UPDATE ON notifications.push_delivery_log
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION notifications.update_notification_delivery_status();
CREATE TRIGGER trigger_notifications_batch_progress
    AFTER UPDATE ON notifications.notifications
    FOR EACH ROW
    WHEN (
        OLD.delivery_status IS DISTINCT FROM NEW.delivery_status
        OR OLD.is_read IS DISTINCT FROM NEW.is_read
    )
    EXECUTE FUNCTION notifications.update_batch_progress();

-- database-dup/triggers/notifications/05_announcements.triggers.sql
CREATE TRIGGER trigger_announcement_views_increment
    AFTER INSERT ON notifications.announcement_views
    FOR EACH ROW
    EXECUTE FUNCTION notifications.increment_announcement_views();
CREATE TRIGGER trigger_announcement_views_clicks
    AFTER UPDATE ON notifications.announcement_views
    FOR EACH ROW
    WHEN (OLD.clicked IS DISTINCT FROM NEW.clicked OR OLD.dismissed IS DISTINCT FROM NEW.dismissed)
    EXECUTE FUNCTION notifications.increment_announcement_clicks();
CREATE TRIGGER trigger_conversation_channels_validate
    BEFORE INSERT OR UPDATE ON notifications.conversation_channels
    FOR EACH ROW
    EXECUTE FUNCTION notifications.validate_conversation_channel();

-- database-dup/triggers/users/01_timestamp.triggers.sql
CREATE TRIGGER trigger_users_profiles_updated_at
    BEFORE UPDATE ON users.profiles
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();
CREATE TRIGGER trigger_users_contacts_updated_at
    BEFORE UPDATE ON users.contacts
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();
CREATE TRIGGER trigger_users_contact_groups_updated_at
    BEFORE UPDATE ON users.contact_groups
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();
CREATE TRIGGER trigger_users_settings_updated_at
    BEFORE UPDATE ON users.settings
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();
CREATE TRIGGER trigger_users_privacy_overrides_updated_at
    BEFORE UPDATE ON users.privacy_overrides
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();
CREATE TRIGGER trigger_users_preferences_updated_at
    BEFORE UPDATE ON users.preferences
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();
CREATE TRIGGER trigger_users_reports_updated_at
    BEFORE UPDATE ON users.reports
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- database-dup/triggers/users/02_user_setup.triggers.sql
CREATE TRIGGER trigger_auth_users_create_profile
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION users.create_default_profile();
CREATE TRIGGER trigger_session_create_device
    AFTER INSERT ON auth.sessions
    FOR EACH ROW
    EXECUTE FUNCTION users.create_device_from_session();
CREATE TRIGGER trigger_auth_users_create_settings
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION users.create_default_settings();

-- database-dup/triggers/users/03_validation.triggers.sql
CREATE TRIGGER trigger_users_profiles_validate_username
    BEFORE INSERT OR UPDATE ON users.profiles
    FOR EACH ROW
    WHEN (NEW.username IS NOT NULL)
    EXECUTE FUNCTION users.validate_username();
CREATE TRIGGER trigger_users_contacts_prevent_self
    BEFORE INSERT OR UPDATE ON users.contacts
    FOR EACH ROW
    EXECUTE FUNCTION users.prevent_self_contact();
CREATE TRIGGER trigger_users_blocked_prevent_self
    BEFORE INSERT ON users.blocked_users
    FOR EACH ROW
    EXECUTE FUNCTION users.prevent_self_blocking();

-- database-dup/triggers/users/04_profile_tracking.triggers.sql
CREATE TRIGGER trigger_users_profiles_log_changes
    AFTER UPDATE ON users.profiles
    FOR EACH ROW
    WHEN (
        OLD.display_name IS DISTINCT FROM NEW.display_name
        OR OLD.bio IS DISTINCT FROM NEW.bio
        OR OLD.avatar_url IS DISTINCT FROM NEW.avatar_url
    )
    EXECUTE FUNCTION users.log_profile_changes();
CREATE TRIGGER trigger_users_contacts_accepted
    BEFORE UPDATE ON users.contacts
    FOR EACH ROW
    WHEN (NEW.status = 'accepted' AND OLD.status != 'accepted')
    EXECUTE FUNCTION users.update_contact_accepted_at();

-- database-dup/triggers/users/05_status_activity.triggers.sql
CREATE TRIGGER trigger_users_status_views_increment
    AFTER INSERT ON users.status_views
    FOR EACH ROW
    EXECUTE FUNCTION users.increment_status_views();

-- database-dup/triggers/users/06_device_management.triggers.sql
CREATE TRIGGER trigger_users_devices_last_active
    BEFORE UPDATE ON users.devices
    FOR EACH ROW
    EXECUTE FUNCTION users.update_device_last_active();
CREATE TRIGGER trigger_users_devices_cleanup
    AFTER INSERT ON users.devices
    FOR EACH ROW
    EXECUTE FUNCTION users.cleanup_old_devices();

-- database-dup/views/analytics_views.sql

