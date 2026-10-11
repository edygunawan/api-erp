-- Defaults for the application settings form; preserve existing configuration.
INSERT INTO setting_groups(id,code,name) VALUES
('00000000-0000-4000-8003-000000000002','GENERAL','General'),
('00000000-0000-4000-8003-000000000003','LOCALIZATION','Localization')
ON CONFLICT DO NOTHING;

INSERT INTO settings(id,group_id,setting_key,setting_value,data_type)
SELECT '00000000-0000-4000-8004-000000000003',id,'application_name','React Admin Kit','STRING' FROM setting_groups WHERE code='GENERAL'
ON CONFLICT DO NOTHING;

INSERT INTO settings(id,group_id,setting_key,setting_value,data_type)
SELECT '00000000-0000-4000-8004-000000000004',id,'description','Business administration application','STRING' FROM setting_groups WHERE code='GENERAL'
ON CONFLICT DO NOTHING;

INSERT INTO settings(id,group_id,setting_key,setting_value,data_type)
SELECT '00000000-0000-4000-8004-000000000005',id,'timezone','Asia/Makassar','STRING' FROM setting_groups WHERE code='LOCALIZATION'
ON CONFLICT DO NOTHING;

INSERT INTO settings(id,group_id,setting_key,setting_value,data_type)
SELECT '00000000-0000-4000-8004-000000000006',id,'date_format','DD/MM/YYYY','STRING' FROM setting_groups WHERE code='LOCALIZATION'
ON CONFLICT DO NOTHING;
