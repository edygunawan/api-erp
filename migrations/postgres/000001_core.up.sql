CREATE TABLE users (
 id UUID PRIMARY KEY,
 full_name VARCHAR(150) NOT NULL CHECK (btrim(full_name) <> ''),
 username VARCHAR(100) NOT NULL CHECK (username ~ '^[A-Za-z0-9_.-]+$'),
 password VARCHAR(255) NOT NULL,
 email VARCHAR(150), phone VARCHAR(30),
 status VARCHAR(8) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE')),
 auth_version BIGINT NOT NULL DEFAULT 0,
 last_login_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 created_by UUID REFERENCES users(id) ON DELETE SET NULL,
 updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX uq_users_username ON users(lower(username));
CREATE UNIQUE INDEX uq_users_email ON users(lower(email)) WHERE email IS NOT NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE TABLE roles (
 id UUID PRIMARY KEY, code VARCHAR(100) NOT NULL UNIQUE CHECK (btrim(code)<>''),
 name VARCHAR(150) NOT NULL CHECK (btrim(name)<>''), description VARCHAR(500), is_system BOOLEAN NOT NULL DEFAULT FALSE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 created_by UUID REFERENCES users(id) ON DELETE SET NULL, updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);
CREATE TABLE permissions (
 id UUID PRIMARY KEY, code VARCHAR(150) NOT NULL UNIQUE CHECK (btrim(code)<>''),
 name VARCHAR(150) NOT NULL CHECK (btrim(name)<>''), description VARCHAR(500), module VARCHAR(100) NOT NULL CHECK (btrim(module)<>''),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 created_by UUID REFERENCES users(id) ON DELETE SET NULL, updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX idx_permissions_module ON permissions(module);
CREATE TABLE user_roles (
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), created_by UUID REFERENCES users(id) ON DELETE SET NULL,
 PRIMARY KEY(user_id,role_id)
);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
CREATE TABLE role_permissions (
 role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
 permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), created_by UUID REFERENCES users(id) ON DELETE SET NULL,
 PRIMARY KEY(role_id,permission_id)
);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);
