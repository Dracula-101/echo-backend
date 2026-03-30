-- =====================================================
-- USERS TRIGGERS — Device Management
-- =====================================================
--
-- Purpose:   Manage the lifecycle of registered user devices:
--              - Auto-update last_active_at on every device update
--              - Deactivate stale devices (90+ days inactive)
--                when a new device is registered
--
-- Depends:   functions/users.trigger_functions.sql
--              -> users.update_device_last_active()
--              -> users.cleanup_old_devices()
--
-- Tables:    users.devices
--
-- =====================================================


-- Bump last_active_at to NOW() on every device update.
-- This ensures the device activity timestamp stays current
-- without requiring the application layer to set it explicitly.
CREATE TRIGGER trigger_users_devices_last_active
    BEFORE UPDATE ON users.devices
    FOR EACH ROW
    EXECUTE FUNCTION users.update_device_last_active();

-- When a new device is registered, deactivate any of the
-- same user's devices that haven't been used in 90 days.
-- This prevents accumulation of stale device records.
CREATE TRIGGER trigger_users_devices_cleanup
    AFTER INSERT ON users.devices
    FOR EACH ROW
    EXECUTE FUNCTION users.cleanup_old_devices();
