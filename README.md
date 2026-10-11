# Go REST API

A reusable Go REST API foundation with PostgreSQL/MySQL persistence, JWT authentication,
refresh sessions, and database-backed role-based access control (RBAC).

The foundation includes only users, authentication, roles, permissions, user-role
assignments, and role-permission assignments. It has no business modules.

## Architecture

```text
cmd/api                 Application entry point
cmd/migrate             Explicit migration command
cmd/bootstrap-admin     Explicit initial administrator command
internal/app            Composition root and application lifecycle
internal/config         Environment configuration
internal/domain         User, role, and permission entities
internal/auth           JWT adapter, identity helpers, core permission codes
internal/database       Driver factory, credential providers and migration lifecycle
internal/usecase        Application rules, DTOs, repository interfaces
internal/repository     PostgreSQL/MySQL adapters, repository factory and transactions
internal/http           Chi routes, middleware, handlers, JSON delivery
internal/helper         UUID generation and field validation
migrations/postgres     PostgreSQL schema and RBAC seeds
migrations/mysql        MySQL schema and RBAC seeds
```

Request flow: router → authentication → permission middleware → handler → usecase
→ repository. Usecases depend on interfaces and do not depend on HTTP, SQL, or
logging. `internal/app` wires these layers. The module remains
`github.com/edygunawan/go-rest-api`.

## Setup

Requires Go 1.25.5 or later, and either PostgreSQL 16 or later or MySQL 8.0.16 or
later. Create a database/user, copy `.env.example` to `.env`, and configure:

- `DB_DRIVER`: `mysql` (default) or `postgres`.
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`: separate connection fields.
  An empty `DB_PORT` selects 3306 for MySQL or 5432 for PostgreSQL.
- `DB_AUTH_MODE`: `password` by default; other modes require an injected credentials provider.
- `MYSQL_TLS_MODE`: `disable` or `verify-full`; applies to MySQL only.
- `POSTGRES_SSL_MODE`: `disable`, `require`, `verify-ca`, or `verify-full`; applies to PostgreSQL only.
- `DB_TLS_CA_FILE`: optional CA certificate bundle for TLS verification.
- `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS`: pool limits, defaults 25 / 10.
- `DB_CONN_MAX_LIFETIME_SECONDS`: connection lifetime, default 180 seconds.
- `JWT_SECRET`: random secret containing at least 32 bytes.
- `JWT_ISSUER`: defaults to `go-rest-api`.
- `JWT_EXPIRE_MINUTES`: access lifetime, default 15 minutes.
- `JWT_REFRESH_EXPIRE_MINUTES`: refresh lifetime, default 10080 minutes (7 days).
- `HTTP_PORT`: defaults to 8080.
- `CORS_ALLOWED_ORIGINS`: optional comma-separated origins.
- `SENTRY_DSN`: optional reporting service.

For example, MySQL uses:

```dotenv
DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=rest_api
DB_USER=rest_api
DB_PASSWORD=<your-password>
```

For PostgreSQL, select `DB_DRIVER=postgres` and set `DB_PORT=5432` (or leave it
empty). Keep the remaining fields appropriate for that database. `DATABASE_URL`
is no longer used by application configuration. Changing the driver selects an
implementation and its migrations; it does not transfer existing database data.

Generate a signing secret with `openssl rand -hex 32` and set it in `.env`.
Environment variables take precedence over `.env`.

Run migrations explicitly:

```sh
go run ./cmd/migrate up
```

The selected driver uses `migrations/mysql/` or `migrations/postgres/`. Both have
the same generic tables, seeds and authentication-version behavior. Migrations
are embedded in the binary, so running from a different working
directory does not change migration discovery. Running `up` again is safe.
`AUTO_MIGRATE=true` optionally runs migrations during API startup; the default is
`false`.

To undo a specific number of migrations:

```sh
go run ./cmd/migrate down 1
```

`down` requires an explicit positive count. Rollback removes the corresponding
schema/data; use it on databases where that removal is intended.

## Initial administrator

Migrations seed an `ADMIN` system role, fifteen core permissions, and a default
active administrator (`admin` / `Admin123!`, bcrypt hash, no email). Change this
password after first login using `PUT /users/{id}`. Migration `000004` skips account
creation when username `admin` or any ADMIN assignment already exists; existing
credentials are preserved. Rolling back `000004` deletes the seeded account and
its sessions/assignments.

For a separate administrator with custom credentials, use bootstrap. Running
bootstrap for the seeded username preserves its existing password.

Set these values in the environment or `.env`, then run the bootstrap command:

```dotenv
ADMIN_FULL_NAME=Administrator
ADMIN_USERNAME=custom_admin
ADMIN_EMAIL=custom_admin@example.com
ADMIN_PASSWORD=<your-password>
```

```sh
go run ./cmd/bootstrap-admin
go run ./cmd/api
```

Email is optional. Name, username, and password are required. Bootstrap creates a
user and assigns `ADMIN` in one transaction. Repeating it for the same active user
preserves the existing profile and password and ensures the role assignment
exists. A conflicting username/email or an inactive existing account is rejected.
Bootstrap is an operator command; it is not exposed as an HTTP endpoint.

## Authentication

`POST /auth/login` accepts:

```json
{"username_or_email":"admin","password":"<your-password>"}
```

A successful login returns:

```json
{
  "access_token":"...",
  "refresh_token":"...",
  "token_type":"Bearer",
  "expires_in":900
}
```

Use `Authorization: Bearer <access_token>` on protected routes. Access and refresh
JWTs have separate audiences and token types. Parsing requires HS256, the expected
issuer/audience, expiry, issued-at time, and valid user/session/token IDs.

`POST /auth/refresh-token` accepts:

```json
{"refresh_token":"..."}
```

Each refresh rotates the stored token hash atomically and returns a new pair.
Only one concurrent request can consume the same refresh token; subsequent reuse
returns 401. Refresh extends the session's refresh expiry. Earlier access tokens
from the session retain their own expiry until the session is revoked.

`POST /auth/logout` revokes the authenticated session, immediately invalidating its
access and refresh tokens. Other login sessions remain active. Password changes
and deactivation invalidate all existing sessions for that user; reactivation
does not restore those sessions.

`GET /auth/me` returns the current profile, role codes, and permission codes.
Authentication checks the current user status and session against PostgreSQL.
Authorization resolves current assignments from PostgreSQL without a permission
cache, so committed assignment changes apply on the next request across instances.

Passwords use bcrypt. Password bytes, including leading/trailing whitespace, are
preserved; length must be 1–72 bytes. User responses never include password hashes.
Expired/revoked sessions remain available for operator maintenance; they cannot
be used for authentication. They can be purged with an operator SQL maintenance
job using `expires_at` / `revoked_at`.

## Routes and permissions

| Method | Route | Access |
| --- | --- | --- |
| GET | `/health` | Public |
| POST | `/auth/login` | Public |
| POST | `/auth/refresh-token` | Valid refresh token |
| GET | `/auth/me` | Authenticated |
| POST | `/auth/logout` | Authenticated |
| GET | `/users`, `/users/{id}` | `USER.READ` |
| POST | `/users` | `USER.CREATE` |
| PUT | `/users/{id}` | `USER.UPDATE` |
| GET | `/roles`, `/roles/{id}` | `ROLE.READ` |
| POST | `/roles` | `ROLE.CREATE` |
| PUT | `/roles/{id}` | `ROLE.UPDATE` |
| DELETE | `/roles/{id}` | `ROLE.DELETE` |
| GET | `/permissions`, `/permissions/{id}` | `PERMISSION.READ` |
| POST | `/permissions` | `PERMISSION.CREATE` |
| PUT | `/permissions/{id}` | `PERMISSION.UPDATE` |
| DELETE | `/permissions/{id}` | `PERMISSION.DELETE` |
| GET | `/settings`, `/settings/{id}`, `/setting-groups`, `/setting-groups/{id}` | `SETTING.READ` |
| POST | `/settings`, `/setting-groups` | `SETTING.CREATE` |
| PUT | `/settings/{id}`, `/setting-groups/{id}` | `SETTING.UPDATE` |
| DELETE | `/settings/{id}`, `/setting-groups/{id}` | `SETTING.DELETE` |

Routes accept trailing slashes. Public registration and physical user deletion
are not included; use `INACTIVE` to disable a user.

Create a user with:

```json
{
  "full_name":"Example User",
  "username":"example",
  "password":"<your-password>",
  "email":"example@example.com",
  "phone":null,
  "status":"ACTIVE",
  "roles":["<role-uuid>"]
}
```

Username and email uniqueness are case-insensitive. Usernames allow letters,
digits, underscores, dots, and hyphens, avoiding ambiguity with email login.
Create defaults to `ACTIVE`. Update requires `full_name` and `username`; omitted
`password` preserves the current password, and omitted `status` preserves the
current status. `roles` contains role UUIDs and replaces the full assignment set;
an empty or omitted array clears assignments. Email/phone are replacement fields.
Duplicate assignment IDs are removed, and entity/assignment changes commit together.

Create/update a role with:

```json
{"code":"READER","name":"Reader","description":null,"permissions":["<permission-uuid>"]}
```

Permission UUIDs replace the full assignment set. An empty or omitted array clears
it. Role/permission codes are normalized to uppercase. `is_system` is read-only,
controlled by seeds; the `ADMIN` role code cannot be renamed, and the role cannot
be deleted. Core permission codes/modules cannot be changed and core permissions
cannot be deleted. Display names/descriptions remain editable.

Create/update a permission with:

```json
{"code":"AUDIT.READ","name":"Read audit","module":"AUDIT","description":null}
```

Custom permissions have ordinary CRUD behavior. `ADMIN` receives the seeded core
permissions through explicit relationships, without a wildcard bypass. Grant new
project permissions by updating the appropriate role's assignment set.

Errors use `{"message":"..."}` with 400 for invalid input, 401 for invalid
credentials/session, 403 for missing permissions/protected operations, 404 for
missing records, 409 for duplicates, and 500 for internal failures. Internal SQL
and infrastructure details are not returned. JSON bodies must contain one object,
have no unknown fields, and fit within 1 MiB. List responses use `{data, meta}`, with empty `data: []`.

## Paginated list queries

**Contract change:** GET `/users`, `/roles`, and `/permissions` now return
`{data: [...], meta: {page, page_size, total, total_pages}}`, replacing raw arrays.
Default page is 1, page size is 20, and the maximum page size is 100. Pagination,
search and filters execute in SQL on PostgreSQL and MySQL. Existing detail,
mutation, auth, and RBAC contracts are unchanged.

Use `?page=2&page_size=20&search=john&sort=full_name&order=asc` for user lists;
user `status` and permission `module` accept exact or comma-separated IN filters.
Invalid/unknown query parameters are rejected with 400. Empty and out-of-range
pages return `data: []` with count metadata; out-of-range requests keep their page.

See [Simple Query Builder](docs/list-queries.md) for endpoint whitelists, repository
usage, security guarantees, limitations, index considerations, and tests.
**Frontend list and assignment-catalog consumers must migrate to this envelope
and fetch the necessary pages before deploying with this API.** Frontend code is
intentionally outside this API-only change.

## Tests

```sh
go build ./...
go test -race ./...
go vet ./...
```

Unit tests cover JWT validation/type separation, password semantics,
authentication failures, middleware, JSON requests, configuration, connection
factory validation, and per-connection credentials callbacks.

Enable the integration suites with both database targets:

```sh
export TEST_POSTGRES_URL='postgres://test_user:test_password@localhost:5432/test_db?sslmode=disable'
export TEST_MYSQL_DSN='test_user:test_password@tcp(127.0.0.1:3306)/test_db?parseTime=true'
go test -race ./...
```

To require both targets and prevent accidental skips in CI:

```sh
go test -race ./internal/repository/factory ./internal/database ./internal/app -args -require-test-databases
```

Flags `-test-postgres-url` and `-test-mysql-dsn` can supply the targets instead of
environment variables. The PostgreSQL account needs permission to create/drop
schemas; the MySQL test account needs permission to create/drop databases. Each
test creates and removes a unique namespace. Supplied base databases are never
migrated or removed. Without targets, the integration suites are skipped while
unit tests still run.

One repository contract suite runs against both factories, checking CRUD,
missing/duplicate records, timestamps, nullable fields, assignment rollback,
permission resolution, cascades, password-version invalidation, refresh replay,
concurrency, and repeat/concurrent admin bootstrap. The existing API regressions
also run on both databases, including migration up/down and auth/RBAC flows.

## Future dynamic credentials and IAM

Database selection and credential selection are separate. `database.Open` builds
a driver connector, and `CredentialsProvider.Credentials(ctx, target)` is called
before **every new physical connection**. Existing pooled connections do not
request another token for each query. Providers must be safe for concurrent calls
and honor context cancellation.

A provider receives the selected driver, exact host/port, database name and user.
It returns a password or a short-lived token. Inject it with:

```go
option := database.WithCredentialsProvider(provider)
db, err := database.Open(ctx, dbConfig, option)
```

The same option is accepted by `app.New`, `app.BootstrapAdmin`,
`database.Migrate`, and `database.MigrateContext`. Keep provider construction in
the composition root and reuse it for API, bootstrap, and migration entry points.
An unknown `DB_AUTH_MODE` without a provider fails rather than falling back to the
static password.

AWS IAM token generation is not included yet. A future AWS implementation can
use the SDK credential chain and RDS token signing inside this interface, without
changing repositories, handlers or usecases. Configure verified TLS with the RDS
CA bundle. For RDS MySQL, return `Credentials{Password: token,
AllowCleartext: true}`; the connector refuses that plugin unless
`MYSQL_TLS_MODE=verify-full`. PostgreSQL tokens use the normal password field with
`POSTGRES_SSL_MODE=verify-full`.

Integration tests force fresh physical connections, verify repeated provider
calls, verify failure/recovery without a static-password fallback, and prove that
migrations and bootstrap use the same provider. MySQL query connections keep
multi-statement execution disabled; only the migration connector enables it.

## Settings API

Settings are **global application configuration**, not per-user preferences.
Migration `000005` creates `setting_groups` and `settings`; `000006` seeds the
APPEARANCE group with `appearance_mode=light` and `color_theme=mono`; `000007`
grants SETTING.READ/CREATE/UPDATE/DELETE to ADMIN. Run `go run ./cmd/migrate up`
before starting the updated API. The new permission codes are protected core
permissions. Both MySQL and PostgreSQL implement the same contracts.

Both `/setting-groups` and `/settings` support POST (201), GET list/detail (200),
PUT detail (200) and DELETE detail (200, `{"message":"settings deleted"}` or
`{"message":"setting-groups deleted"}`). IDs in paths and group references are
UUIDs. PUT replaces the editable fields; it is not PATCH. Deleting a group also
deletes all of its settings through the existing foreign-key cascade.

Create/update a group:

```json
{"code":"APPEARANCE","name":"Appearance"}
```

Codes are trimmed, uppercased, required and limited to 50 characters. Names are
trimmed, required and limited to 100 characters. Codes are unique.

Create/update a setting:

```json
{
  "group_id":"00000000-0000-4000-8003-000000000001",
  "setting_key":"appearance_mode",
  "setting_value":"dark",
  "data_type":"STRING",
  "description":"Default appearance mode"
}
```

Keys are trimmed/lowercased, required, limited to 100 characters and unique
**within a group**. `group_id` must reference an existing group. `data_type`
defaults to STRING and is normalized to uppercase; accepted values are STRING,
NUMBER, BOOLEAN and JSON. `setting_value` is a **string or null**, including for
NUMBER/BOOLEAN/JSON: `"12.5"`, `"true"`, or `"{\"enabled\":true}"`.
STRING preserves whitespace; other types trim whitespace and must contain a
finite JSON number, exactly true/false, or valid JSON respectively. Values are
limited to 65,535 UTF-8 bytes for parity with MySQL TEXT. Null/omitted values are
stored as SQL NULL. Descriptions are nullable, trimmed and limited to 255
characters; blank descriptions become null. Omitted optional fields in PUT are
cleared, rather than preserved. Appearance values use ordinary STRING semantics;
the API does not maintain a frontend palette catalog.

A setting response contains `id`, `group_id`, `setting_key`, `setting_value`,
`data_type`, `description`, `created_at` and `updated_at`. A group response contains
`id`, `code`, `name`, `created_at` and `updated_at`. Timestamps are server-maintained.

Lists use the existing `{ "data": [], "meta": { "page": 1, "page_size": 20,
"total": 0, "total_pages": 0 } }` envelope and existing bounded
`page/page_size/search/sort/order` parameters. Empty results encode as an array.
Filters are exact, with comma-separated values supported:

| Resource | Search | Filters | Sort fields |
| --- | --- | --- | --- |
| `/settings` | key, description | `group_id`, `group_code`, `setting_key`, `data_type` | `id`, `setting_key`, `data_type`, `created_at`, `updated_at` |
| `/setting-groups` | code, name | `code` | `id`, `code`, `name`, `created_at`, `updated_at` |

Default sorting is ascending setting_key/code, with UUID tie-breaking. For example,
`GET /settings?group_code=APPEARANCE&setting_key=appearance_mode` returns the current
appearance default for a client that has SETTING.READ. To change it, use its ID and
full editable fields with `PUT /settings/{id}`.

Unknown JSON fields and query parameters, invalid typed values, invalid UUIDs and
invalid references return 400; missing authentication returns 401; missing
permissions return 403; missing records return 404; duplicate codes or group/key
pairs return 409. Errors retain the existing JSON error format. Permission checks
use live backend RBAC assignments; there is no ADMIN bypass.

The frontend currently persists appearance locally. These endpoints are ready for
integration but do not change frontend behavior by themselves. Updating settings
is global, not a private user-specific appearance change.

## Application settings form

Migration `000008` adds GENERAL (`application_name`, `description`) and
LOCALIZATION (`timezone`, `date_format`) defaults while preserving existing
values. Run `go run ./cmd/migrate up` before using the Settings page.

`GET /settings/application` requires APPLICATIONSETTING.READ and returns:

```json
{"application_name":"React Admin Kit","description":"Business administration application","timezone":"Asia/Makassar","date_format":"DD/MM/YYYY"}
```

`PUT /settings/application` requires APPLICATIONSETTING.UPDATE and accepts the same complete
object. It validates all four fields before updating their STRING setting values
in **one transaction**, returning the normalized saved object. A missing record
returns 404 and rolls back all edits; it does not silently recreate deleted
configuration. Missing/invalid configuration on GET produces an explicit error.
Application name is required (150 characters), description is optional (500
characters), timezone must be a valid IANA name (100 characters; Local is rejected),
and date format must be DD/MM/YYYY, MM/DD/YYYY or YYYY-MM-DD. Unknown fields and
invalid values return 400. IANA timezone data is included in the binary.

The values are global application configuration; this endpoint does not change
personal appearance preferences, database UTC behavior, or existing date rendering.

Migration 000009 grants ADMIN APPLICATIONSETTING.READ and APPLICATIONSETTING.UPDATE. These govern only the application aggregate endpoints; SETTING permissions still govern individual settings and groups. Reapply explicit grants for custom roles as needed.

Settings responses from GET /settings, GET /settings/{id}, POST /settings and
PUT /settings/{id} now include a nested `group` object with `id`, `code`,
`name`, `created_at` and `updated_at`, alongside the existing `group_id`.
The repository selects this relation through the same many-to-one SQL JOIN used
for group filtering; pagination and totals are unchanged. Write payloads remain
unchanged and do not accept a nested group object. The query builder continues to
own safe WHERE/ORDER/LIMIT/OFFSET clauses while repositories own SELECT/JOIN and
typed relation mapping. No database migration is needed for this response change.

Users list/detail and create/update responses now include `role_details`, an array
of `{id, code, name}` summaries. Existing `roles` UUID arrays and write payloads
remain unchanged; `/auth/me` continues returning role codes. Role summaries are
loaded by joining roles into the existing batch query for the selected user page,
without per-user lookups or expanding the parent pagination query. Users without
assignments return an empty array. Only basic role identity is exposed, not role
permissions or other management data. No migration is required.
