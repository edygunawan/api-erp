# PostgreSQL foundation migrations

- `000001`: users, roles, permissions, and their assignment tables, constraints and indexes.
- `000002`: generic ADMIN role, eleven core permissions, and explicit role-permission grants.
- `000003`: hashed refresh sessions and a password/status trigger that increments the user's authentication version.

- `000004`: default active `admin` user with ADMIN role; initial password `Admin123!` (bcrypt). Existing admin accounts are preserved.

Set `DB_DRIVER=postgres` and run `go run ./cmd/migrate up` from the project directory. Files are embedded in the
binary. Use `go run ./cmd/migrate down <steps>` for an explicit rollback count.
Application startup migrates only when `AUTO_MIGRATE=true`.

Change the default password after first login. Use `cmd/bootstrap-admin` for a
separate administrator with custom credentials. See the root README for details.

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
