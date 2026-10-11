CREATE TABLE setting_groups (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 code VARCHAR(50) NOT NULL UNIQUE CHECK(btrim(code)<>''),
 name VARCHAR(100) NOT NULL CHECK(btrim(name)<>''),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE settings (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 group_id UUID NOT NULL REFERENCES setting_groups(id) ON DELETE CASCADE,
 setting_key VARCHAR(100) NOT NULL CHECK(btrim(setting_key)<>''),
 setting_value TEXT,
 data_type VARCHAR(7) NOT NULL DEFAULT 'STRING' CHECK(data_type IN ('STRING','NUMBER','BOOLEAN','JSON')),
 description VARCHAR(255),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 CONSTRAINT uq_settings_group_key UNIQUE(group_id,setting_key)
);
