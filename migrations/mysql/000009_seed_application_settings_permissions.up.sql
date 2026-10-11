-- Application configuration has independent read/update access.
INSERT INTO permissions(id,code,name,module) VALUES
('00000000-0000-4000-8001-000000000016','APPLICATIONSETTING.READ','Read application settings','APPLICATIONSETTING'),
('00000000-0000-4000-8001-000000000017','APPLICATIONSETTING.UPDATE','Update application settings','APPLICATIONSETTING')
ON DUPLICATE KEY UPDATE id = permissions.id;

INSERT INTO role_permissions(role_id,permission_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permissions p
WHERE r.code='ADMIN' AND p.code IN ('APPLICATIONSETTING.READ','APPLICATIONSETTING.UPDATE')
ON DUPLICATE KEY UPDATE role_id = role_permissions.role_id;
