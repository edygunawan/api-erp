CREATE TABLE users (
 id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
 full_name VARCHAR(150) NOT NULL,
 username VARCHAR(100) NOT NULL,
 username_normalized VARCHAR(100) GENERATED ALWAYS AS (LOWER(username)) STORED,
 password VARCHAR(255) NOT NULL,
 email VARCHAR(150),
 email_normalized VARCHAR(150) GENERATED ALWAYS AS (LOWER(email)) STORED,
 phone VARCHAR(30), status VARCHAR(8) NOT NULL DEFAULT 'ACTIVE',
 auth_version BIGINT NOT NULL DEFAULT 0,
 last_login_at DATETIME(6),
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 created_by CHAR(36) CHARACTER SET ascii COLLATE ascii_bin,
 updated_by CHAR(36) CHARACTER SET ascii COLLATE ascii_bin,
 CONSTRAINT uq_users_username UNIQUE(username_normalized),
 CONSTRAINT uq_users_email UNIQUE(email_normalized),
 CONSTRAINT ck_users_full_name CHECK (TRIM(full_name)<>''),
 CONSTRAINT ck_users_username CHECK (username REGEXP '^[A-Za-z0-9_.-]+$'),
 CONSTRAINT ck_users_status CHECK (status IN ('ACTIVE','INACTIVE')),
 CONSTRAINT fk_users_created_by FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE SET NULL,
 CONSTRAINT fk_users_updated_by FOREIGN KEY(updated_by) REFERENCES users(id) ON DELETE SET NULL,
 INDEX idx_users_status(status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
CREATE TABLE roles (
 id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
 code VARCHAR(100) NOT NULL UNIQUE,
 name VARCHAR(150) NOT NULL, description VARCHAR(500), is_system BOOLEAN NOT NULL DEFAULT FALSE,
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6), updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 created_by CHAR(36) CHARACTER SET ascii COLLATE ascii_bin,
 updated_by CHAR(36) CHARACTER SET ascii COLLATE ascii_bin,
 CONSTRAINT ck_roles_code CHECK(TRIM(code)<>''),
 CONSTRAINT ck_roles_name CHECK(TRIM(name)<>''),
 CONSTRAINT fk_roles_created_by FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE SET NULL,
 CONSTRAINT fk_roles_updated_by FOREIGN KEY(updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
CREATE TABLE permissions (
 id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL PRIMARY KEY,
 code VARCHAR(150) NOT NULL UNIQUE,
 name VARCHAR(150) NOT NULL, description VARCHAR(500), module VARCHAR(100) NOT NULL,
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6), updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 created_by CHAR(36) CHARACTER SET ascii COLLATE ascii_bin,
 updated_by CHAR(36) CHARACTER SET ascii COLLATE ascii_bin,
 CONSTRAINT ck_permissions_code CHECK(TRIM(code)<>''),
 CONSTRAINT ck_permissions_name CHECK(TRIM(name)<>''),
 CONSTRAINT ck_permissions_module CHECK(TRIM(module)<>''),
 CONSTRAINT fk_permissions_created_by FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE SET NULL,
 CONSTRAINT fk_permissions_updated_by FOREIGN KEY(updated_by) REFERENCES users(id) ON DELETE SET NULL,
 INDEX idx_permissions_module(module)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
CREATE TABLE user_roles (
 user_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 role_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 created_by CHAR(36) CHARACTER SET ascii COLLATE ascii_bin,
 PRIMARY KEY(user_id,role_id),
 CONSTRAINT fk_user_roles_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
 CONSTRAINT fk_user_roles_role FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE,
 CONSTRAINT fk_user_roles_created_by FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE SET NULL,
 INDEX idx_user_roles_role(role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
CREATE TABLE role_permissions (
 role_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 permission_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 created_by CHAR(36) CHARACTER SET ascii COLLATE ascii_bin,
 PRIMARY KEY(role_id,permission_id),
 CONSTRAINT fk_role_permissions_role FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE,
 CONSTRAINT fk_role_permissions_permission FOREIGN KEY(permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
 CONSTRAINT fk_role_permissions_created_by FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE SET NULL,
 INDEX idx_role_permissions_permission(permission_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
