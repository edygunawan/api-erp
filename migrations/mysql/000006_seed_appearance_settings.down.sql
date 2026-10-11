-- Remove only migration-owned records. Keep a group containing other settings.
DELETE FROM settings
WHERE group_id IN (SELECT id FROM setting_groups WHERE code = 'APPEARANCE')
 AND ((id = '00000000-0000-4000-8004-000000000001' AND setting_key = 'appearance_mode')
   OR (id = '00000000-0000-4000-8004-000000000002' AND setting_key = 'color_theme'));

DELETE FROM setting_groups
WHERE id = '00000000-0000-4000-8003-000000000001' AND code = 'APPEARANCE'
 AND NOT EXISTS (SELECT 1 FROM settings WHERE settings.group_id = setting_groups.id);
