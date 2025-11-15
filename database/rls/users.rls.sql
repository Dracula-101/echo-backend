-- =====================================================
-- USERS SCHEMA - ROW LEVEL SECURITY (RLS)
-- =====================================================

-- Enable RLS on all users tables
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

-- =====================================================
-- PROFILES TABLE POLICIES
-- =====================================================

-- Users can view public profiles
CREATE POLICY profiles_select_public
    ON users.profiles
    FOR SELECT
    USING (
        profile_visibility = 'public'
        AND search_visibility = TRUE
        AND deactivated_at IS NULL
    );

-- Users can view their own profile
CREATE POLICY profiles_select_own
    ON users.profiles
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can view friends' profiles
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

-- Users can update their own profile
CREATE POLICY profiles_update_own
    ON users.profiles
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- Admins can view all profiles
CREATE POLICY profiles_select_admin
    ON users.profiles
    FOR SELECT
    USING (auth.is_admin());

-- =====================================================
-- CONTACTS TABLE POLICIES
-- =====================================================

-- Users can view their own contacts
CREATE POLICY contacts_select_own
    ON users.contacts
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can insert their own contacts
CREATE POLICY contacts_insert_own
    ON users.contacts
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

-- Users can update their own contacts
CREATE POLICY contacts_update_own
    ON users.contacts
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- Users can delete their own contacts
CREATE POLICY contacts_delete_own
    ON users.contacts
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Users can see incoming contact requests
CREATE POLICY contacts_select_incoming
    ON users.contacts
    FOR SELECT
    USING (contact_user_id = auth.current_user_id());

-- =====================================================
-- CONTACT GROUPS TABLE POLICIES
-- =====================================================

-- Users can manage their own contact groups
CREATE POLICY contact_groups_all_own
    ON users.contact_groups
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- SETTINGS TABLE POLICIES
-- =====================================================

-- Users can manage their own settings
CREATE POLICY settings_all_own
    ON users.settings
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- BLOCKED USERS TABLE POLICIES
-- =====================================================

-- Users can view their own blocks
CREATE POLICY blocked_users_select_own
    ON users.blocked_users
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can block others
CREATE POLICY blocked_users_insert_own
    ON users.blocked_users
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

-- Users can update their blocks (unblock)
CREATE POLICY blocked_users_update_own
    ON users.blocked_users
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- Users can delete their blocks
CREATE POLICY blocked_users_delete_own
    ON users.blocked_users
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- =====================================================
-- PRIVACY OVERRIDES TABLE POLICIES
-- =====================================================

-- Users can manage their own privacy overrides
CREATE POLICY privacy_overrides_all_own
    ON users.privacy_overrides
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- STATUS HISTORY TABLE POLICIES
-- =====================================================

-- Users can view public statuses
CREATE POLICY status_history_select_public
    ON users.status_history
    FOR SELECT
    USING (
        privacy = 'public'
        AND expires_at > NOW()
        AND deleted_at IS NULL
    );

-- Users can view their own statuses
CREATE POLICY status_history_select_own
    ON users.status_history
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can view contacts' statuses
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

-- Users can manage their own statuses
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

-- =====================================================
-- STATUS VIEWS TABLE POLICIES
-- =====================================================

-- Users can view their own status views (who viewed their status)
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

-- Users can insert views when viewing statuses
CREATE POLICY status_views_insert
    ON users.status_views
    FOR INSERT
    WITH CHECK (viewer_user_id = auth.current_user_id());

-- =====================================================
-- ACTIVITY LOG TABLE POLICIES
-- =====================================================

-- Users can view their own activity log
CREATE POLICY activity_log_select_own
    ON users.activity_log
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Service can insert activity logs
CREATE POLICY activity_log_insert_service
    ON users.activity_log
    FOR INSERT
    WITH CHECK (TRUE);

-- Admins can view all activity logs
CREATE POLICY activity_log_select_admin
    ON users.activity_log
    FOR SELECT
    USING (auth.is_admin());

-- =====================================================
-- PREFERENCES TABLE POLICIES
-- =====================================================

-- Users can manage their own preferences
CREATE POLICY preferences_all_own
    ON users.preferences
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- DEVICES TABLE POLICIES
-- =====================================================

-- Users can view their own devices
CREATE POLICY devices_select_own
    ON users.devices
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can insert their own devices
CREATE POLICY devices_insert_own
    ON users.devices
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

-- Users can update their own devices
CREATE POLICY devices_update_own
    ON users.devices
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- Users can delete their own devices
CREATE POLICY devices_delete_own
    ON users.devices
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- =====================================================
-- ACHIEVEMENTS TABLE POLICIES
-- =====================================================

-- Users can view their own achievements
CREATE POLICY achievements_select_own
    ON users.achievements
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can view public achievements of others
CREATE POLICY achievements_select_public
    ON users.achievements
    FOR SELECT
    USING (
        display_on_profile = TRUE
        AND is_unlocked = TRUE
    );

-- Service can insert achievements
CREATE POLICY achievements_insert_service
    ON users.achievements
    FOR INSERT
    WITH CHECK (TRUE);

-- Users can update display settings of their achievements
CREATE POLICY achievements_update_display
    ON users.achievements
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (
        user_id = auth.current_user_id()
    );

-- =====================================================
-- REPORTS TABLE POLICIES
-- =====================================================

-- Users can view reports they created
CREATE POLICY reports_select_reporter
    ON users.reports
    FOR SELECT
    USING (reporter_user_id = auth.current_user_id());

-- Users can create reports
CREATE POLICY reports_insert
    ON users.reports
    FOR INSERT
    WITH CHECK (reporter_user_id = auth.current_user_id());

-- Admins can view and manage all reports
CREATE POLICY reports_admin_all
    ON users.reports
    FOR ALL
    USING (auth.is_admin());

-- Moderators can view assigned reports
CREATE POLICY reports_select_assigned
    ON users.reports
    FOR SELECT
    USING (assigned_to = auth.current_user_id());