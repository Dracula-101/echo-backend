-- =====================================================
-- USERS TRIGGERS — Input Validation
-- =====================================================
--
-- Purpose:   Enforce business rules and data integrity on
--            user-facing tables. Prevents:
--              - Invalid username formats
--              - Self-contact (adding yourself as a contact)
--              - Self-blocking (blocking yourself)
--
-- Depends:   functions/users.trigger_functions.sql
--              -> users.validate_username()
--              -> users.prevent_self_contact()
--              -> users.prevent_self_blocking()
--
-- Tables:    users.profiles       (username validation)
--            users.contacts       (self-contact prevention)
--            users.blocked_users  (self-block prevention)
--
-- Note:      These are all BEFORE triggers that can reject
--            the operation by raising an exception.
--
-- =====================================================


-- Validate username format on INSERT and UPDATE:
--   - Must be 3-50 characters
--   - Only alphanumeric and underscores allowed
--   - Automatically lowercased for consistency
CREATE TRIGGER trigger_users_profiles_validate_username
    BEFORE INSERT OR UPDATE ON users.profiles
    FOR EACH ROW
    WHEN (NEW.username IS NOT NULL)
    EXECUTE FUNCTION users.validate_username();

-- Prevent a user from adding themselves as a contact.
-- The CHECK constraint on the table provides a backup,
-- but this trigger gives a clearer error message.
CREATE TRIGGER trigger_users_contacts_prevent_self
    BEFORE INSERT OR UPDATE ON users.contacts
    FOR EACH ROW
    EXECUTE FUNCTION users.prevent_self_contact();

-- Prevent a user from blocking themselves.
-- The CHECK constraint on the table provides a backup,
-- but this trigger gives a clearer error message.
CREATE TRIGGER trigger_users_blocked_prevent_self
    BEFORE INSERT ON users.blocked_users
    FOR EACH ROW
    EXECUTE FUNCTION users.prevent_self_blocking();
