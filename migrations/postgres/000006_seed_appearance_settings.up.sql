-- Application defaults only; preserve any existing group and configured values.
INSERT INTO setting_groups (id, code, name)
VALUES ('00000000-0000-4000-8003-000000000001', 'APPEARANCE', 'Appearance')
ON CONFLICT DO NOTHING;

INSERT INTO settings (id, group_id, setting_key, setting_value, data_type, description)
SELECT '00000000-0000-4000-8004-000000000001', id, 'appearance_mode', 'light', 'STRING', 'Default appearance mode: light, dark or system'
FROM setting_groups WHERE code = 'APPEARANCE'
ON CONFLICT DO NOTHING;

INSERT INTO settings (id, group_id, setting_key, setting_value, data_type, description)
SELECT '00000000-0000-4000-8004-000000000002', id, 'color_theme', 'mono', 'STRING', 'Default color theme: mono, indigo, blue, emerald, violet or rose'
FROM setting_groups WHERE code = 'APPEARANCE'
ON CONFLICT DO NOTHING;
