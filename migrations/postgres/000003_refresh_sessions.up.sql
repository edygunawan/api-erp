CREATE TABLE refresh_sessions (
 id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 token_hash CHAR(64) NOT NULL UNIQUE, token_version BIGINT NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL, revoked_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_refresh_sessions_user ON refresh_sessions(user_id);
CREATE INDEX idx_refresh_sessions_expiry ON refresh_sessions(expires_at);
-- Sessions created before a password change/deactivation cannot become valid again.
CREATE FUNCTION bump_auth_version() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.password IS DISTINCT FROM OLD.password OR (NEW.status='INACTIVE' AND OLD.status='ACTIVE') THEN
  NEW.auth_version := OLD.auth_version+1;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER users_auth_version BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION bump_auth_version();
