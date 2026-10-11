CREATE TABLE refresh_sessions (
 id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
 user_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 token_version BIGINT NOT NULL, expires_at DATETIME(6) NOT NULL, revoked_at DATETIME(6),
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6), updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 CONSTRAINT fk_refresh_sessions_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
 INDEX idx_refresh_sessions_user(user_id), INDEX idx_refresh_sessions_expiry(expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
-- One SET expression avoids client-specific DELIMITER directives.
CREATE TRIGGER users_auth_version BEFORE UPDATE ON users FOR EACH ROW
SET NEW.auth_version = OLD.auth_version + IF(
 NOT (NEW.password <=> OLD.password) OR (NEW.status='INACTIVE' AND OLD.status='ACTIVE'),1,0
);
