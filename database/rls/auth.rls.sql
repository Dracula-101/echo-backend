CREATE OR REPLACE FUNCTION auth.current_user_id()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_user_id', TRUE)::UUID;
EXCEPTION
    WHEN OTHERS THEN RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION auth.is_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM auth.users
        WHERE id              = auth.current_user_id()
          AND account_status  = 'active'
          AND metadata->>'role' = 'admin'
    );
EXCEPTION
    WHEN OTHERS THEN RETURN FALSE;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;


ALTER TABLE auth.users                     ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.sessions                  ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.otp_verifications         ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.oauth_providers           ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.password_reset_tokens     ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.email_verification_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.security_events           ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.login_history             ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.api_keys                  ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.outbox                    ENABLE ROW LEVEL SECURITY;


-- auth.users
CREATE POLICY users_select_own
    ON auth.users FOR SELECT
    USING (id = auth.current_user_id());

CREATE POLICY users_select_admin
    ON auth.users FOR SELECT
    USING (auth.is_admin());

CREATE POLICY users_update_own
    ON auth.users FOR UPDATE
    USING (id = auth.current_user_id())
    WITH CHECK (id = auth.current_user_id());

CREATE POLICY users_update_admin
    ON auth.users FOR UPDATE
    USING (auth.is_admin());

CREATE POLICY users_insert_service
    ON auth.users FOR INSERT
    WITH CHECK (TRUE);

CREATE POLICY users_delete_admin
    ON auth.users FOR DELETE
    USING (auth.is_admin());


-- auth.sessions
CREATE POLICY sessions_select_own
    ON auth.sessions FOR SELECT
    USING (user_id = auth.current_user_id());

CREATE POLICY sessions_update_own
    ON auth.sessions FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

CREATE POLICY sessions_insert_service
    ON auth.sessions FOR INSERT
    WITH CHECK (TRUE);

CREATE POLICY sessions_delete_own
    ON auth.sessions FOR DELETE
    USING (user_id = auth.current_user_id());

CREATE POLICY sessions_admin_all
    ON auth.sessions FOR ALL
    USING (auth.is_admin());


-- auth.otp_verifications
CREATE POLICY otp_service_all
    ON auth.otp_verifications FOR ALL
    USING (TRUE);

CREATE POLICY otp_select_own
    ON auth.otp_verifications FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR identifier IN (
            SELECT email        FROM auth.users WHERE id = auth.current_user_id()
            UNION
            SELECT phone_number FROM auth.users WHERE id = auth.current_user_id()
        )
    );


-- auth.oauth_providers
CREATE POLICY oauth_select_own
    ON auth.oauth_providers FOR SELECT
    USING (user_id = auth.current_user_id());

CREATE POLICY oauth_delete_own
    ON auth.oauth_providers FOR DELETE
    USING (user_id = auth.current_user_id());

CREATE POLICY oauth_service_all
    ON auth.oauth_providers FOR ALL
    USING (TRUE);


-- auth.password_reset_tokens
CREATE POLICY password_reset_service_all
    ON auth.password_reset_tokens FOR ALL
    USING (TRUE);

CREATE POLICY password_reset_select_own
    ON auth.password_reset_tokens FOR SELECT
    USING (user_id = auth.current_user_id());


-- auth.email_verification_tokens
CREATE POLICY email_verify_service_all
    ON auth.email_verification_tokens FOR ALL
    USING (TRUE);


-- auth.security_events
CREATE POLICY security_events_select_own
    ON auth.security_events FOR SELECT
    USING (user_id = auth.current_user_id());

CREATE POLICY security_events_select_admin
    ON auth.security_events FOR SELECT
    USING (auth.is_admin());

CREATE POLICY security_events_insert_service
    ON auth.security_events FOR INSERT
    WITH CHECK (TRUE);


-- auth.login_history
CREATE POLICY login_history_select_own
    ON auth.login_history FOR SELECT
    USING (user_id = auth.current_user_id());

CREATE POLICY login_history_select_admin
    ON auth.login_history FOR SELECT
    USING (auth.is_admin());

CREATE POLICY login_history_insert_service
    ON auth.login_history FOR INSERT
    WITH CHECK (TRUE);


-- auth.api_keys
CREATE POLICY api_keys_select_own
    ON auth.api_keys FOR SELECT
    USING (user_id = auth.current_user_id());

CREATE POLICY api_keys_insert_own
    ON auth.api_keys FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

CREATE POLICY api_keys_update_own
    ON auth.api_keys FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

CREATE POLICY api_keys_delete_own
    ON auth.api_keys FOR DELETE
    USING (user_id = auth.current_user_id());

CREATE POLICY api_keys_admin_all
    ON auth.api_keys FOR ALL
    USING (auth.is_admin());

CREATE POLICY api_keys_service_own
    ON auth.api_keys FOR SELECT
    USING (service_name IS NOT NULL);


-- auth.outbox
CREATE POLICY outbox_service_all
    ON auth.outbox FOR ALL
    USING (TRUE);