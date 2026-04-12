SELECT cron.schedule('cleanup-expired-otp',             '0 * * * *',   $$DELETE FROM auth.otp_verifications WHERE expires_at < NOW() - INTERVAL '24 hours'$$);
SELECT cron.schedule('cleanup-expired-pwd-reset',       '0 */6 * * *', $$DELETE FROM auth.password_reset_tokens WHERE expires_at < NOW() - INTERVAL '7 days'$$);
SELECT cron.schedule('cleanup-expired-email-verify',    '0 */6 * * *', $$DELETE FROM auth.email_verification_tokens WHERE expires_at < NOW() - INTERVAL '7 days'$$);
SELECT cron.schedule('mark-expired-sessions',           '0 3 * * *',   $$UPDATE auth.sessions SET revoked_at = NOW(), revoked_reason = 'expired' WHERE expires_at < NOW() AND revoked_at IS NULL$$);
SELECT cron.schedule('unlock-accounts',                 '*/5 * * * *', $$UPDATE auth.users SET account_locked_until = NULL WHERE account_locked_until IS NOT NULL AND account_locked_until < NOW()$$);
SELECT cron.schedule('archive-security-events',         '0 2 * * 0',   $$DELETE FROM auth.security_events WHERE created_at < NOW() - INTERVAL '90 days'$$);
SELECT cron.schedule('archive-login-history',           '0 2 1 * *',   $$DELETE FROM auth.login_history WHERE created_at < NOW() - INTERVAL '180 days'$$);
SELECT cron.schedule('cleanup-auth-outbox',             '*/10 * * * *',$$DELETE FROM auth.outbox WHERE status = 'processed' AND processed_at < NOW() - INTERVAL '24 hours'$$);

SELECT cron.schedule('expire-status-history',           '*/30 * * * *',$$UPDATE users.status_history SET deleted_at = NOW() WHERE expires_at < NOW() AND deleted_at IS NULL$$);
SELECT cron.schedule('hard-delete-old-statuses',        '0 4 * * *',   $$DELETE FROM users.status_history WHERE deleted_at < NOW() - INTERVAL '7 days'$$);
SELECT cron.schedule('purge-activity-log',              '0 3 * * 0',   $$DELETE FROM users.activity_log WHERE created_at < NOW() - INTERVAL '180 days'$$);
SELECT cron.schedule('deactivate-old-devices',          '0 3 * * *',   $$UPDATE users.devices SET is_active = FALSE WHERE last_active_at < NOW() - INTERVAL '90 days' AND is_active = TRUE$$);
SELECT cron.schedule('cleanup-declined-contacts',       '0 3 1 * *',   $$DELETE FROM users.contacts WHERE status = 'declined' AND updated_at < NOW() - INTERVAL '90 days'$$);
SELECT cron.schedule('cleanup-users-outbox',            '*/10 * * * *',$$DELETE FROM users.outbox WHERE status = 'processed' AND processed_at < NOW() - INTERVAL '24 hours'$$);

SELECT cron.schedule('expire-disappearing-messages',    '* * * * *',   $$SELECT messages.expire_disappearing_messages()$$);
SELECT cron.schedule('cleanup-typing-indicators',       '*/5 * * * *', $$SELECT messages.cleanup_expired_typing_indicators()$$);

SELECT cron.schedule('cleanup-deleted-media',           '0 */4 * * *', $$SELECT media.cleanup_deleted_files()$$);

SELECT cron.schedule('cleanup-expired-notifications',   '0 * * * *',   $$SELECT notifications.cleanup_expired_notifications()$$);
