-- -- =====================================================
-- -- AUDIT TRIGGERS - Comprehensive Change Tracking
-- -- =====================================================

-- -- Create audit schema
-- CREATE SCHEMA IF NOT EXISTS audit;

-- -- =====================================================
-- -- AUDIT TABLES
-- -- =====================================================

-- -- Generic audit log for all table changes
-- CREATE TABLE audit.record_changes (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     -- Table information
--     schema_name VARCHAR(100) NOT NULL,
--     table_name VARCHAR(100) NOT NULL,
--     record_id UUID,
    
--     -- Operation details
--     operation VARCHAR(10) NOT NULL, -- INSERT, UPDATE, DELETE
    
--     -- Change data
--     old_data JSONB,
--     new_data JSONB,
--     changed_fields TEXT[], -- Array of changed column names
    
--     -- User context
--     user_id UUID,
--     session_id UUID,
--     ip_address INET,
--     user_agent TEXT,
    
--     -- Timing
--     transaction_id BIGINT DEFAULT txid_current(),
--     operation_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
--     -- Additional context
--     application_name VARCHAR(255),
--     query_text TEXT,
    
--     created_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- -- Sensitive data access log
-- CREATE TABLE audit.sensitive_data_access (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     -- Access details
--     accessed_table VARCHAR(100) NOT NULL,
--     accessed_column VARCHAR(100),
--     record_id UUID,
--     access_type VARCHAR(50), -- SELECT, UPDATE, DELETE
    
--     -- User context
--     user_id UUID NOT NULL,
--     session_id UUID,
--     ip_address INET,
    
--     -- Purpose (optional, can be set by application)
--     access_reason TEXT,
    
--     -- Data accessed (masked for security)
--     data_preview TEXT, -- First/last few chars only
    
--     accessed_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- -- Authentication audit log
-- CREATE TABLE audit.auth_events (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     user_id UUID,
--     email VARCHAR(255),
--     phone_number VARCHAR(20),
    
--     -- Event details
--     event_type VARCHAR(100) NOT NULL, -- login, logout, failed_login, password_change, etc.
--     event_status VARCHAR(50), -- success, failure, blocked, suspicious
    
--     -- Security context
--     ip_address INET,
--     user_agent TEXT,
--     device_id VARCHAR(255),
--     device_fingerprint TEXT,
--     location_country VARCHAR(100),
--     location_city VARCHAR(100),
    
--     -- Authentication details
--     auth_method VARCHAR(50), -- password, oauth, otp, biometric
--     failure_reason TEXT,
    
--     -- Risk assessment
--     risk_level VARCHAR(20), -- low, medium, high, critical
--     risk_factors JSONB,
    
--     -- MFA details
--     mfa_used BOOLEAN DEFAULT FALSE,
--     mfa_method VARCHAR(50),
    
--     created_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- -- Data export audit log
-- CREATE TABLE audit.data_exports (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     user_id UUID NOT NULL,
--     export_type VARCHAR(100), -- full_data, messages, media, contacts
    
--     -- Export details
--     tables_included TEXT[],
--     record_count INTEGER,
--     file_size_bytes BIGINT,
--     file_format VARCHAR(50), -- json, csv, pdf
    
--     -- Status
--     status VARCHAR(50) DEFAULT 'initiated', -- initiated, processing, completed, failed
--     download_url TEXT,
--     download_expires_at TIMESTAMPTZ,
--     downloaded_at TIMESTAMPTZ,
    
--     -- Context
--     ip_address INET,
--     user_agent TEXT,
    
--     initiated_at TIMESTAMPTZ DEFAULT NOW(),
--     completed_at TIMESTAMPTZ
-- );

-- -- Data deletion audit log
-- CREATE TABLE audit.data_deletions (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     user_id UUID,
--     deleted_by_user_id UUID,
    
--     -- Deletion details
--     deletion_type VARCHAR(100), -- soft_delete, hard_delete, anonymization, account_closure
--     table_name VARCHAR(100),
--     record_id UUID,
--     record_count INTEGER DEFAULT 1,
    
--     -- Backup information
--     backup_location TEXT,
--     recovery_deadline TIMESTAMPTZ,
    
--     -- Reason
--     deletion_reason TEXT,
--     is_gdpr_request BOOLEAN DEFAULT FALSE,
--     is_user_requested BOOLEAN DEFAULT FALSE,
    
--     -- Context
--     ip_address INET,
    
--     deleted_at TIMESTAMPTZ DEFAULT NOW(),
--     permanent_deletion_at TIMESTAMPTZ
-- );

-- -- Admin actions audit log
-- CREATE TABLE audit.admin_actions (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     admin_user_id UUID NOT NULL,
--     target_user_id UUID,
    
--     -- Action details
--     action_type VARCHAR(100) NOT NULL, -- ban_user, unban_user, delete_content, grant_role, etc.
--     action_category VARCHAR(50), -- moderation, administration, support
    
--     -- Changes
--     affected_table VARCHAR(100),
--     affected_record_id UUID,
--     old_value JSONB,
--     new_value JSONB,
    
--     -- Justification
--     reason TEXT NOT NULL,
--     notes TEXT,
    
--     -- Approval (for critical actions)
--     requires_approval BOOLEAN DEFAULT FALSE,
--     approved_by_user_id UUID,
--     approved_at TIMESTAMPTZ,
    
--     -- Context
--     ip_address INET,
    
--     created_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- -- Permission changes audit log
-- CREATE TABLE audit.permission_changes (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     user_id UUID NOT NULL,
--     changed_by_user_id UUID,
    
--     -- Permission details
--     permission_type VARCHAR(100), -- conversation_role, admin_grant, feature_access
--     resource_type VARCHAR(100), -- conversation, system, feature
--     resource_id UUID,
    
--     -- Changes
--     old_permission VARCHAR(100),
--     new_permission VARCHAR(100),
--     old_role VARCHAR(50),
--     new_role VARCHAR(50),
    
--     -- Reason
--     change_reason TEXT,
    
--     changed_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- -- Privacy settings changes
-- CREATE TABLE audit.privacy_changes (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     user_id UUID NOT NULL,
    
--     -- Setting details
--     setting_name VARCHAR(100) NOT NULL,
--     setting_category VARCHAR(50), -- profile, messaging, location, data
    
--     -- Changes
--     old_value TEXT,
--     new_value TEXT,
    
--     -- Context
--     ip_address INET,
--     device_id VARCHAR(255),
    
--     changed_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- -- Payment and billing audit
-- CREATE TABLE audit.billing_events (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     user_id UUID NOT NULL,
    
--     -- Transaction details
--     event_type VARCHAR(100), -- purchase, refund, subscription_start, subscription_cancel
--     transaction_id VARCHAR(255),
    
--     -- Amounts
--     amount DECIMAL(10,2),
--     currency VARCHAR(10),
    
--     -- Payment details
--     payment_method VARCHAR(100),
--     payment_provider VARCHAR(100),
    
--     -- Status
--     status VARCHAR(50),
    
--     -- Context
--     ip_address INET,
    
--     created_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- -- Content moderation audit
-- CREATE TABLE audit.moderation_actions (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     moderator_user_id UUID NOT NULL,
--     target_user_id UUID,
    
--     -- Content details
--     content_type VARCHAR(100), -- message, media, profile, status
--     content_id UUID,
    
--     -- Action taken
--     action_type VARCHAR(100), -- flag, hide, delete, warn, ban
--     action_reason VARCHAR(255),
--     violation_type VARCHAR(100),
    
--     -- AI moderation
--     is_auto_moderated BOOLEAN DEFAULT FALSE,
--     ai_confidence_score DECIMAL(5,2),
    
--     -- Review
--     requires_review BOOLEAN DEFAULT FALSE,
--     reviewed_by_user_id UUID,
--     review_status VARCHAR(50),
--     reviewed_at TIMESTAMPTZ,
    
--     -- Appeals
--     is_appealed BOOLEAN DEFAULT FALSE,
--     appeal_status VARCHAR(50),
    
--     created_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- -- API usage audit
-- CREATE TABLE audit.api_calls (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
--     -- API details
--     api_key_id UUID,
--     user_id UUID,
--     endpoint VARCHAR(255) NOT NULL,
--     http_method VARCHAR(20),
    
--     -- Request
--     request_headers JSONB,
--     request_body JSONB,
--     query_params JSONB,
    
--     -- Response
--     status_code INTEGER,
--     response_time_ms INTEGER,
--     response_size_bytes INTEGER,
    
--     -- Rate limiting
--     rate_limit_remaining INTEGER,
--     rate_limit_reset TIMESTAMPTZ,
    
--     -- Context
--     ip_address INET,
--     user_agent TEXT,
    
--     -- Errors
--     is_error BOOLEAN DEFAULT FALSE,
--     error_message TEXT,
    
--     called_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- -- =====================================================
-- -- AUDIT TRIGGER FUNCTIONS
-- -- =====================================================

-- -- Generic audit function for all tables
-- CREATE OR REPLACE FUNCTION audit.log_record_change()
-- RETURNS TRIGGER AS $$
-- DECLARE
--     old_data_json JSONB;
--     new_data_json JSONB;
--     changed_fields TEXT[];
--     excluded_columns TEXT[] := ARRAY['updated_at', 'last_activity_at'];
-- BEGIN
--     -- Convert old and new rows to JSONB
--     IF TG_OP = 'DELETE' THEN
--         old_data_json := to_jsonb(OLD);
--         new_data_json := NULL;
--     ELSIF TG_OP = 'INSERT' THEN
--         old_data_json := NULL;
--         new_data_json := to_jsonb(NEW);
--     ELSE -- UPDATE
--         old_data_json := to_jsonb(OLD);
--         new_data_json := to_jsonb(NEW);
        
--         -- Identify changed fields
--         SELECT array_agg(key)
--         INTO changed_fields
--         FROM jsonb_each(new_data_json)
--         WHERE NOT (key = ANY(excluded_columns))
--         AND (old_data_json->key IS DISTINCT FROM new_data_json->key);
--     END IF;
    
--     -- Insert audit record
--     INSERT INTO audit.record_changes (
--         schema_name,
--         table_name,
--         record_id,
--         operation,
--         old_data,
--         new_data,
--         changed_fields,
--         user_id,
--         ip_address,
--         application_name
--     ) VALUES (
--         TG_TABLE_SCHEMA,
--         TG_TABLE_NAME,
--         COALESCE(NEW.id, OLD.id),
--         TG_OP,
--         old_data_json,
--         new_data_json,
--         changed_fields,
--         auth.current_user_id(),
--         inet_client_addr(),
--         current_setting('application_name', true)
--     );
    
--     RETURN COALESCE(NEW, OLD);
-- END;
-- $$ LANGUAGE plpgsql SECURITY DEFINER;

-- -- Audit function for sensitive data access
-- CREATE OR REPLACE FUNCTION audit.log_sensitive_access()
-- RETURNS TRIGGER AS $$
-- BEGIN
--     -- Log access to sensitive columns
--     IF TG_OP = 'SELECT' THEN
--         INSERT INTO audit.sensitive_data_access (
--             accessed_table,
--             record_id,
--             access_type,
--             user_id,
--             ip_address
--         ) VALUES (
--             TG_TABLE_NAME,
--             NEW.id,
--             'SELECT',
--             auth.current_user_id(),
--             inet_client_addr()
--         );
--     END IF;
    
--     RETURN NEW;
-- END;
-- $$ LANGUAGE plpgsql SECURITY DEFINER;

-- -- Audit function for authentication events
-- CREATE OR REPLACE FUNCTION audit.log_auth_event()
-- RETURNS TRIGGER AS $$
-- DECLARE
--     event_type_val VARCHAR(100);
--     event_status_val VARCHAR(50);
--     user_id_val UUID;
-- BEGIN
--     IF TG_OP = 'INSERT' THEN
--         -- Handle different table structures
--         IF TG_TABLE_NAME = 'users' THEN
--             -- auth.users table has 'id' not 'user_id'
--             event_type_val := 'user_registration';
--             event_status_val := 'success';
--             user_id_val := NEW.id;
            
--             INSERT INTO audit.auth_events (
--                 user_id,
--                 event_type,
--                 event_status,
--                 ip_address,
--                 user_agent
--             ) VALUES (
--                 user_id_val,
--                 event_type_val,
--                 event_status_val,
--                 NEW.created_by_ip,
--                 NEW.created_by_user_agent
--             );
--         ELSIF TG_TABLE_NAME = 'sessions' THEN
--             event_type_val := 'login';
--             event_status_val := 'success';
--             user_id_val := NEW.user_id;
            
--             INSERT INTO audit.auth_events (
--                 user_id,
--                 event_type,
--                 event_status,
--                 ip_address,
--                 user_agent,
--                 device_id
--             ) VALUES (
--                 user_id_val,
--                 event_type_val,
--                 event_status_val,
--                 NEW.ip_address,
--                 NEW.user_agent,
--                 NEW.device_id
--             );
--         ELSIF TG_TABLE_NAME = 'login_history' THEN
--             event_type_val := 'login_attempt';
--             event_status_val := NEW.status;
--             user_id_val := NEW.user_id;
            
--             INSERT INTO audit.auth_events (
--                 user_id,
--                 event_type,
--                 event_status,
--                 ip_address,
--                 user_agent,
--                 device_id
--             ) VALUES (
--                 user_id_val,
--                 event_type_val,
--                 event_status_val,
--                 NEW.ip_address,
--                 NEW.user_agent,
--                 NEW.device_id
--             );
--         END IF;
--     ELSIF TG_OP = 'UPDATE' THEN
--         -- Password change
--         IF TG_TABLE_NAME = 'users' AND OLD.password_hash != NEW.password_hash THEN
--             INSERT INTO audit.auth_events (
--                 user_id,
--                 event_type,
--                 event_status,
--                 ip_address
--             ) VALUES (
--                 NEW.id,  -- auth.users has 'id' not 'user_id'
--                 'password_change',
--                 'success',
--                 inet_client_addr()
--             );
--         END IF;
        
--         -- 2FA changes
--         IF TG_TABLE_NAME = 'users' AND OLD.two_factor_enabled != NEW.two_factor_enabled THEN
--             INSERT INTO audit.auth_events (
--                 user_id,
--                 event_type,
--                 event_status,
--                 mfa_used
--             ) VALUES (
--                 NEW.id,  -- auth.users has 'id' not 'user_id'
--                 CASE WHEN NEW.two_factor_enabled THEN '2fa_enabled' ELSE '2fa_disabled' END,
--                 'success',
--                 NEW.two_factor_enabled
--             );
--         END IF;
--     END IF;
    
--     RETURN NEW;
-- EXCEPTION
--     WHEN OTHERS THEN
--         -- Log error but don't fail the operation
--         RAISE WARNING 'Failed to log auth event for table %: %', TG_TABLE_NAME, SQLERRM;
--         RETURN NEW;
-- END;
-- $$ LANGUAGE plpgsql SECURITY DEFINER;

-- -- Audit function for admin actions
-- CREATE OR REPLACE FUNCTION audit.log_admin_action()
-- RETURNS TRIGGER AS $$
-- DECLARE
--     action_type_val VARCHAR(100);
-- BEGIN
--     -- Determine action type based on changes
--     IF TG_TABLE_NAME = 'users' THEN
--         IF OLD.account_status != NEW.account_status THEN
--             action_type_val := CASE NEW.account_status
--                 WHEN 'suspended' THEN 'suspend_user'
--                 WHEN 'banned' THEN 'ban_user'
--                 WHEN 'active' THEN 'activate_user'
--                 ELSE 'change_user_status'
--             END;
            
--             INSERT INTO audit.admin_actions (
--                 admin_user_id,
--                 target_user_id,
--                 action_type,
--                 action_category,
--                 affected_table,
--                 affected_record_id,
--                 old_value,
--                 new_value,
--                 reason
--             ) VALUES (
--                 auth.current_user_id(),
--                 NEW.id,
--                 action_type_val,
--                 'moderation',
--                 TG_TABLE_NAME,
--                 NEW.id,
--                 jsonb_build_object('account_status', OLD.account_status),
--                 jsonb_build_object('account_status', NEW.account_status),
--                 'Account status changed'
--             );
--         END IF;
--     ELSIF TG_TABLE_NAME = 'conversation_participants' THEN
--         IF OLD.role != NEW.role THEN
--             INSERT INTO audit.permission_changes (
--                 user_id,
--                 changed_by_user_id,
--                 permission_type,
--                 resource_type,
--                 resource_id,
--                 old_role,
--                 new_role,
--                 change_reason
--             ) VALUES (
--                 NEW.user_id,
--                 auth.current_user_id(),
--                 'conversation_role',
--                 'conversation',
--                 NEW.conversation_id,
--                 OLD.role,
--                 NEW.role,
--                 'Role changed in conversation'
--             );
--         END IF;
--     END IF;
    
--     RETURN NEW;
-- END;
-- $$ LANGUAGE plpgsql SECURITY DEFINER;

-- -- Audit function for data deletions
-- CREATE OR REPLACE FUNCTION audit.log_data_deletion()
-- RETURNS TRIGGER AS $$
-- BEGIN
--     INSERT INTO audit.data_deletions (
--         user_id,
--         deleted_by_user_id,
--         deletion_type,
--         table_name,
--         record_id,
--         deletion_reason,
--         is_user_requested,
--         ip_address
--     ) VALUES (
--         COALESCE(OLD.user_id, OLD.uploader_user_id, OLD.sender_user_id),
--         auth.current_user_id(),
--         CASE 
--             WHEN OLD.deleted_at IS NOT NULL THEN 'soft_delete'
--             ELSE 'hard_delete'
--         END,
--         TG_TABLE_NAME,
--         OLD.id,
--         'Record deleted',
--         auth.current_user_id() = COALESCE(OLD.user_id, OLD.uploader_user_id, OLD.sender_user_id),
--         inet_client_addr()
--     );
    
--     RETURN OLD;
-- END;
-- $$ LANGUAGE plpgsql SECURITY DEFINER;

-- -- Audit function for privacy setting changes
-- CREATE OR REPLACE FUNCTION audit.log_privacy_change()
-- RETURNS TRIGGER AS $$
-- DECLARE
--     setting_name_val VARCHAR(100);
--     old_val TEXT;
--     new_val TEXT;
-- BEGIN
--     IF TG_TABLE_NAME = 'settings' THEN
--         -- Log significant privacy changes
--         IF OLD.profile_visibility IS DISTINCT FROM NEW.profile_visibility THEN
--             INSERT INTO audit.privacy_changes (user_id, setting_name, setting_category, old_value, new_value, ip_address)
--             VALUES (NEW.user_id, 'profile_visibility', 'profile', OLD.profile_visibility, NEW.profile_visibility, inet_client_addr());
--         END IF;
        
--         IF OLD.last_seen_visibility IS DISTINCT FROM NEW.last_seen_visibility THEN
--             INSERT INTO audit.privacy_changes (user_id, setting_name, setting_category, old_value, new_value, ip_address)
--             VALUES (NEW.user_id, 'last_seen_visibility', 'profile', OLD.last_seen_visibility, NEW.last_seen_visibility, inet_client_addr());
--         END IF;
        
--         IF OLD.read_receipts_enabled IS DISTINCT FROM NEW.read_receipts_enabled THEN
--             INSERT INTO audit.privacy_changes (user_id, setting_name, setting_category, old_value, new_value, ip_address)
--             VALUES (NEW.user_id, 'read_receipts_enabled', 'messaging', OLD.read_receipts_enabled::TEXT, NEW.read_receipts_enabled::TEXT, inet_client_addr());
--         END IF;
--     END IF;
    
--     RETURN NEW;
-- END;
-- $$ LANGUAGE plpgsql SECURITY DEFINER;

-- -- Audit function for moderation actions
-- CREATE OR REPLACE FUNCTION audit.log_moderation_action()
-- RETURNS TRIGGER AS $$
-- BEGIN
--     IF TG_TABLE_NAME = 'message_reports' THEN
--         INSERT INTO audit.moderation_actions (
--             moderator_user_id,
--             target_user_id,
--             content_type,
--             content_id,
--             action_type,
--             action_reason,
--             violation_type
--         ) VALUES (
--             NEW.reporter_user_id,
--             (SELECT sender_user_id FROM messages.messages WHERE id = NEW.message_id),
--             'message',
--             NEW.message_id,
--             'flag',
--             NEW.description,
--             NEW.report_type
--         );
--     END IF;
    
--     RETURN NEW;
-- END;
-- $$ LANGUAGE plpgsql SECURITY DEFINER;

-- -- =====================================================
-- -- APPLY AUDIT TRIGGERS
-- -- =====================================================

-- -- Auth schema auditing
-- CREATE TRIGGER audit_users_changes
--     AFTER INSERT OR UPDATE OR DELETE ON auth.users
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- CREATE TRIGGER audit_auth_events
--     AFTER INSERT OR UPDATE ON auth.users
--     FOR EACH ROW EXECUTE FUNCTION audit.log_auth_event();

-- CREATE TRIGGER audit_sessions_auth
--     AFTER INSERT ON auth.sessions
--     FOR EACH ROW EXECUTE FUNCTION audit.log_auth_event();

-- CREATE TRIGGER audit_admin_actions_users
--     AFTER UPDATE ON auth.users
--     FOR EACH ROW 
--     WHEN (auth.is_admin() AND OLD.account_status IS DISTINCT FROM NEW.account_status)
--     EXECUTE FUNCTION audit.log_admin_action();

-- -- Users schema auditing
-- CREATE TRIGGER audit_profiles_changes
--     AFTER INSERT OR UPDATE OR DELETE ON users.profiles
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- CREATE TRIGGER audit_contacts_changes
--     AFTER INSERT OR UPDATE OR DELETE ON users.contacts
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- CREATE TRIGGER audit_blocked_users_changes
--     AFTER INSERT OR UPDATE OR DELETE ON users.blocked_users
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- CREATE TRIGGER audit_settings_privacy
--     AFTER UPDATE ON users.settings
--     FOR EACH ROW EXECUTE FUNCTION audit.log_privacy_change();

-- -- Messages schema auditing
-- CREATE TRIGGER audit_messages_changes
--     AFTER INSERT OR UPDATE OR DELETE ON messages.messages
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- CREATE TRIGGER audit_messages_deletion
--     BEFORE DELETE ON messages.messages
--     FOR EACH ROW EXECUTE FUNCTION audit.log_data_deletion();

-- CREATE TRIGGER audit_conversations_changes
--     AFTER INSERT OR UPDATE OR DELETE ON messages.conversations
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- CREATE TRIGGER audit_participants_permission_changes
--     AFTER UPDATE ON messages.conversation_participants
--     FOR EACH ROW 
--     WHEN (OLD.role IS DISTINCT FROM NEW.role)
--     EXECUTE FUNCTION audit.log_admin_action();

-- CREATE TRIGGER audit_message_reports
--     AFTER INSERT ON messages.message_reports
--     FOR EACH ROW EXECUTE FUNCTION audit.log_moderation_action();

-- -- Media schema auditing
-- CREATE TRIGGER audit_files_changes
--     AFTER INSERT OR UPDATE OR DELETE ON media.files
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- CREATE TRIGGER audit_files_deletion
--     BEFORE DELETE ON media.files
--     FOR EACH ROW EXECUTE FUNCTION audit.log_data_deletion();

-- -- Notifications schema auditing
-- CREATE TRIGGER audit_notifications_changes
--     AFTER INSERT OR UPDATE ON notifications.notifications
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- -- Location schema auditing
-- CREATE TRIGGER audit_location_changes
--     AFTER INSERT OR UPDATE OR DELETE ON location.user_locations
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- CREATE TRIGGER audit_location_shares_changes
--     AFTER INSERT OR UPDATE OR DELETE ON location.location_shares
--     FOR EACH ROW EXECUTE FUNCTION audit.log_record_change();

-- -- =====================================================
-- -- AUDIT INDEXES FOR PERFORMANCE
-- -- =====================================================

-- CREATE INDEX idx_record_changes_table ON audit.record_changes(schema_name, table_name);
-- CREATE INDEX idx_record_changes_record_id ON audit.record_changes(record_id);
-- CREATE INDEX idx_record_changes_user ON audit.record_changes(user_id);
-- CREATE INDEX idx_record_changes_timestamp ON audit.record_changes(operation_timestamp);
-- CREATE INDEX idx_record_changes_operation ON audit.record_changes(operation);

-- CREATE INDEX idx_sensitive_access_user ON audit.sensitive_data_access(user_id);
-- CREATE INDEX idx_sensitive_access_table ON audit.sensitive_data_access(accessed_table);
-- CREATE INDEX idx_sensitive_access_timestamp ON audit.sensitive_data_access(accessed_at);

-- CREATE INDEX idx_auth_events_user ON audit.auth_events(user_id);
-- CREATE INDEX idx_auth_events_type ON audit.auth_events(event_type);
-- CREATE INDEX idx_auth_events_timestamp ON audit.auth_events(created_at);
-- CREATE INDEX idx_auth_events_ip ON audit.auth_events(ip_address);

-- CREATE INDEX idx_data_exports_user ON audit.data_exports(user_id);
-- CREATE INDEX idx_data_exports_status ON audit.data_exports(status);

-- CREATE INDEX idx_data_deletions_user ON audit.data_deletions(user_id);
-- CREATE INDEX idx_data_deletions_table ON audit.data_deletions(table_name);
-- CREATE INDEX idx_data_deletions_timestamp ON audit.data_deletions(deleted_at);

-- CREATE INDEX idx_admin_actions_admin ON audit.admin_actions(admin_user_id);
-- CREATE INDEX idx_admin_actions_target ON audit.admin_actions(target_user_id);
-- CREATE INDEX idx_admin_actions_type ON audit.admin_actions(action_type);
-- CREATE INDEX idx_admin_actions_timestamp ON audit.admin_actions(created_at);

-- CREATE INDEX idx_permission_changes_user ON audit.permission_changes(user_id);
-- CREATE INDEX idx_permission_changes_timestamp ON audit.permission_changes(changed_at);

-- CREATE INDEX idx_privacy_changes_user ON audit.privacy_changes(user_id);
-- CREATE INDEX idx_privacy_changes_timestamp ON audit.privacy_changes(changed_at);

-- CREATE INDEX idx_moderation_actions_moderator ON audit.moderation_actions(moderator_user_id);
-- CREATE INDEX idx_moderation_actions_target ON audit.moderation_actions(target_user_id);
-- CREATE INDEX idx_moderation_actions_timestamp ON audit.moderation_actions(created_at);

-- -- =====================================================
-- -- AUDIT RETENTION & CLEANUP FUNCTIONS
-- -- =====================================================

-- -- Function to archive old audit logs
-- CREATE OR REPLACE FUNCTION audit.archive_old_logs(days_to_keep INTEGER DEFAULT 90)
-- RETURNS void AS $$
-- BEGIN
--     -- Move old records to archive tables (implement based on your archival strategy)
--     -- For now, we'll just delete very old records
    
--     DELETE FROM audit.record_changes
--     WHERE operation_timestamp < NOW() - (days_to_keep || ' days')::INTERVAL;
    
--     DELETE FROM audit.sensitive_data_access
--     WHERE accessed_at < NOW() - (days_to_keep || ' days')::INTERVAL;
    
--     -- Keep auth events longer (365 days)
--     DELETE FROM audit.auth_events
--     WHERE created_at < NOW() - INTERVAL '365 days';
    
--     -- Keep admin actions indefinitely (don't delete)
--     -- Keep moderation actions indefinitely (don't delete)
    
--     RAISE NOTICE 'Audit logs older than % days have been archived', days_to_keep;
-- END;
-- $$ LANGUAGE plpgsql;

-- -- Function to get audit trail for a specific record
-- CREATE OR REPLACE FUNCTION audit.get_record_history(
--     p_schema_name VARCHAR,
--     p_table_name VARCHAR,
--     p_record_id UUID
-- )
-- RETURNS TABLE (
--     operation VARCHAR,
--     changed_fields TEXT[],
--     changed_by UUID,
--     changed_at TIMESTAMPTZ,
--     old_data JSONB,
--     new_data JSONB
-- ) AS $$
-- BEGIN
--     RETURN QUERY
--     SELECT 
--         rc.operation,
--         rc.changed_fields,
--         rc.user_id,
--         rc.operation_timestamp,
--         rc.old_data,
--         rc.new_data
--     FROM audit.record_changes rc
--     WHERE rc.schema_name = p_schema_name
--     AND rc.table_name = p_table_name
--     AND rc.record_id = p_record_id
--     ORDER BY rc.operation_timestamp DESC;
-- END;
-- $$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- -- Grant permissions on audit schema
-- GRANT USAGE ON SCHEMA audit TO app_user;
-- GRANT SELECT ON ALL TABLES IN SCHEMA audit TO app_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA audit TO app_user;