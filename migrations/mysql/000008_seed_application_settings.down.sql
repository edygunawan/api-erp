-- Remove only migration-owned defaults; keep groups containing other settings.
DELETE FROM settings WHERE
(id='00000000-0000-4000-8004-000000000003' AND setting_key='application_name' AND group_id IN (SELECT id FROM setting_groups WHERE code='GENERAL'))
 OR (id='00000000-0000-4000-8004-000000000004' AND setting_key='description' AND group_id IN (SELECT id FROM setting_groups WHERE code='GENERAL'))
 OR (id='00000000-0000-4000-8004-000000000005' AND setting_key='timezone' AND group_id IN (SELECT id FROM setting_groups WHERE code='LOCALIZATION'))
 OR (id='00000000-0000-4000-8004-000000000006' AND setting_key='date_format' AND group_id IN (SELECT id FROM setting_groups WHERE code='LOCALIZATION'));

DELETE FROM setting_groups WHERE
((id='00000000-0000-4000-8003-000000000002' AND code='GENERAL') OR (id='00000000-0000-4000-8003-000000000003' AND code='LOCALIZATION'))
AND NOT EXISTS (SELECT 1 FROM settings WHERE settings.group_id=setting_groups.id);
