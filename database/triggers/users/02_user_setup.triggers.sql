-- =====================================================
-- USERS TRIGGERS — User Registration Setup
-- =====================================================
--
-- Purpose:   Automatically provision default records in
--            the users schema when a new user is created
--            in auth.users. Every new user gets:
--              - A starter profile (users.profiles)
--              - A device record (users.devices) to associate sessions with a device even before they log in from it
--              - Default settings (users.settings)
--
-- Depends:   functions/users.trigger_functions.sql
--              -> users.create_default_profile()
--              -> users.create_device_from_session()
--              -> users.create_default_settings()
--
-- Tables:    auth.users    (source — fires on INSERT)
--            users.profiles, users.devices, users.settings
--            (targets — rows created)
--
-- Note:      These triggers fire on auth.users, not on
--            users schema tables. They establish the
--            cross-schema dependency between auth and users.
--
-- =====================================================


-- Create a starter profile with a generated username
-- (user_ + first 8 chars of UUID) for every new user.
CREATE TRIGGER trigger_auth_users_create_profile
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION users.create_default_profile();

-- Create a device record with the user_id for every new user. This
-- allows us to associate sessions with a device even if the user logs in
-- from a new device that doesn't exist in users.devices yet.
CREATE TRIGGER trigger_session_create_device
    AFTER INSERT ON auth.sessions
    FOR EACH ROW
    EXECUTE FUNCTION users.create_device_from_session();

-- Create a default settings row with all columns at their
-- DEFAULT values for every new user.
CREATE TRIGGER trigger_auth_users_create_settings
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION users.create_default_settings();
