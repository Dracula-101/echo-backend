-- =====================================================
-- USERS TRIGGERS — Timestamp Management
-- =====================================================
--
-- Purpose:   Automatically maintain `updated_at` columns
--            on users schema tables whenever rows are modified.
--
-- Depends:   functions/users.trigger_functions.sql
--              -> users.update_updated_at_column()
--
-- Tables:    users.profiles
--            users.contacts
--            users.contact_groups
--            users.settings
--            users.privacy_overrides
--            users.preferences
--            users.reports
--
-- =====================================================


-- Keep updated_at current on user profiles
CREATE TRIGGER trigger_users_profiles_updated_at
    BEFORE UPDATE ON users.profiles
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Keep updated_at current on contacts (e.g. status changes, mute)
CREATE TRIGGER trigger_users_contacts_updated_at
    BEFORE UPDATE ON users.contacts
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Keep updated_at current on contact groups
CREATE TRIGGER trigger_users_contact_groups_updated_at
    BEFORE UPDATE ON users.contact_groups
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Keep updated_at current on user settings
CREATE TRIGGER trigger_users_settings_updated_at
    BEFORE UPDATE ON users.settings
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Keep updated_at current on per-contact privacy overrides
CREATE TRIGGER trigger_users_privacy_overrides_updated_at
    BEFORE UPDATE ON users.privacy_overrides
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Keep updated_at current on user key-value preferences
CREATE TRIGGER trigger_users_preferences_updated_at
    BEFORE UPDATE ON users.preferences
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();

-- Keep updated_at current on user reports
CREATE TRIGGER trigger_users_reports_updated_at
    BEFORE UPDATE ON users.reports
    FOR EACH ROW
    EXECUTE FUNCTION users.update_updated_at_column();
