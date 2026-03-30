-- =====================================================
-- USERS TRIGGERS — User Registration Setup
-- =====================================================
--
-- Purpose:   Automatically provision default records in
--            the users schema when a new user is created
--            in auth.users. Every new user gets:
--              - A starter profile (users.profiles)
--              - A placeholder device (users.devices)
--              - Default settings (users.settings)
--
-- Depends:   functions/users.trigger_functions.sql
--              -> users.create_default_profile()
--              -> users.create_default_device()
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

-- Create a placeholder device record for every new user.
-- The real device info is registered via users.register_device()
-- on first login from a device.
CREATE TRIGGER trigger_auth_users_create_device
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION users.create_default_device();

-- Create a default settings row with all columns at their
-- DEFAULT values for every new user.
CREATE TRIGGER trigger_auth_users_create_settings
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION users.create_default_settings();
