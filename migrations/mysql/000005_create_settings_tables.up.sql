CREATE TABLE setting_groups (
 id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT (UUID()) PRIMARY KEY,
 code VARCHAR(50) NOT NULL,
 name VARCHAR(100) NOT NULL,
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 CONSTRAINT uq_setting_groups_code UNIQUE(code),
 CONSTRAINT ck_setting_groups_code CHECK(TRIM(code)<>''),
 CONSTRAINT ck_setting_groups_name CHECK(TRIM(name)<>'')
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE settings (
 id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT (UUID()) PRIMARY KEY,
 group_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 setting_key VARCHAR(100) NOT NULL,
 setting_value TEXT,
 data_type VARCHAR(7) NOT NULL DEFAULT 'STRING',
 description VARCHAR(255),
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 CONSTRAINT uq_settings_group_key UNIQUE(group_id,setting_key),
 CONSTRAINT ck_settings_key CHECK(TRIM(setting_key)<>''),
 CONSTRAINT ck_settings_data_type CHECK(data_type IN ('STRING','NUMBER','BOOLEAN','JSON')),
 CONSTRAINT fk_settings_group FOREIGN KEY(group_id) REFERENCES setting_groups(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
