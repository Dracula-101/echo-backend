-- =====================================================
-- USERS TRIGGERS — Profile & Contact Tracking
-- =====================================================
--
-- Purpose:   Track changes to user profiles and contact
--            relationships for activity logging and state
--            management:
--              - Log profile field changes (display_name,
--                bio, avatar_url) to users.activity_log
--              - Record the accepted_at timestamp when a
--                contact request is accepted
--
-- Depends:   functions/users.trigger_functions.sql
--              -> users.log_profile_changes()
--              -> users.update_contact_accepted_at()
--            functions/users.functions.sql
--              -> users.log_activity()  (utility)
--
-- Tables:    users.profiles  (change logging)
--            users.contacts  (acceptance tracking)
--
-- =====================================================


-- Log changes to key profile fields (display_name, bio,
-- avatar_url) into users.activity_log. Only fires when
-- at least one of these tracked fields actually changes.
CREATE TRIGGER trigger_users_profiles_log_changes
    AFTER UPDATE ON users.profiles
    FOR EACH ROW
    WHEN (
        OLD.display_name IS DISTINCT FROM NEW.display_name
        OR OLD.bio IS DISTINCT FROM NEW.bio
        OR OLD.avatar_url IS DISTINCT FROM NEW.avatar_url
    )
    EXECUTE FUNCTION users.log_profile_changes();

-- Set the accepted_at timestamp when a contact request
-- transitions to 'accepted' status.
CREATE TRIGGER trigger_users_contacts_accepted
    BEFORE UPDATE ON users.contacts
    FOR EACH ROW
    WHEN (NEW.status = 'accepted' AND OLD.status != 'accepted')
    EXECUTE FUNCTION users.update_contact_accepted_at();
