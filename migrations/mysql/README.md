# MySQL foundation migrations

Requires MySQL 8.0.16 or later and InnoDB. The schema mirrors the PostgreSQL core:
users, roles, permissions, assignments, and refresh sessions. Lowercase generated
columns enforce case-insensitive username/email uniqueness, while binary columns
preserve permission codes and password hashes. Connections use UTC for dates.

- `000001`: core tables, foreign keys, checks and indexes.
- `000002`: ADMIN role, eleven core permissions, and explicit grants.
- `000003`: hashed refresh sessions and the authentication-version trigger.

- `000004`: default active `admin` user with ADMIN role; initial password `Admin123!` (bcrypt). Existing admin accounts are preserved.

Set `DB_DRIVER=mysql` and run `go run ./cmd/migrate up`. Use
`go run ./cmd/migrate down <steps>` for an explicit rollback. Files are embedded;
no DELIMITER command is required by the trigger migration. Only the migration
connector enables multi-statement execution. Change the default password after first login.

- `000005`: `setting_groups` and `settings`, UUID defaults, unique group codes and
  group/key pairs, data types STRING/NUMBER/BOOLEAN/JSON, and cascading group deletion.
- `000006`: APPEARANCE group with `appearance_mode=light` and `color_theme=mono`
  (STRING). Existing configured values are preserved. Seed rollback removes only
  migration-owned records and retains groups that contain other settings.

These are application defaults, not per-user preferences. Settings CRUD endpoints are protected by SETTING permissions; frontend
synchronization is not introduced. Updated timestamps follow the core tables:
writers must update `updated_at` when changing a record. Type metadata describes the
value; the settings usecase validates typed values and updates timestamps.

- `000007`: SETTING.READ/CREATE/UPDATE/DELETE permissions, granted explicitly to
  ADMIN for both settings and setting-group endpoints. Core permission protection
  prevents changing these codes/modules or deleting them through the API.

- `000008`: GENERAL and LOCALIZATION defaults for the atomic application settings
  form endpoint. Existing values are preserved; rollback removes owned seed rows.
