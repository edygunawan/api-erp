-- Default login: admin / Admin123! Change the password after first login.
-- Preserve existing admin accounts and credentials.
INSERT INTO users (id, full_name, username, password, status)
SELECT '00000000-0000-4000-8002-000000000001', 'Administrator', 'admin', '$2a$10$1qHclxJQUMXiu0kGuOvxVOtbdBp770OOdT/J0GyabUkCwL81A9gPS', 'ACTIVE'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE LOWER(username) = 'admin' OR id = '00000000-0000-4000-8002-000000000001')
  AND NOT EXISTS (
    SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE r.code = 'ADMIN'
  );

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u CROSS JOIN roles r
WHERE u.id = '00000000-0000-4000-8002-000000000001' AND r.code = 'ADMIN'
  AND NOT EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_id = r.id);
