# Simple Query Builder

Use the Simple Query Builder for ordinary list endpoints. Use explicit repository
SQL for complex queries and reports. This helper is not an ORM or universal SQL
builder: it constructs WHERE/ORDER BY/LIMIT/OFFSET suffixes, without fetching,
scanning models, mapping relations, generating joins, or interpreting SQL.

## Standard API contract

**Breaking change:** `GET /users`, `GET /roles`, and `GET /permissions` now return
an object, not the previous raw array, even when no query parameters are supplied:

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 0,
    "total_pages": 0
  }
}
```

`total` counts matching records before pagination. `total_pages` is the ceiling of
`total / page_size`, and is 0 for no matches. Requested pages beyond the last page
return HTTP 200, empty `data: []`, and unchanged requested page/total metadata.
There is no legacy-array switch or second pagination representation. Individual
DTO fields, authentication endpoints, authorization codes, and mutation contracts
are unchanged.

Frontend consumers must adapt both lists and assignment catalogs: read `data`
and `meta`, fetch SQL pages, and send the feature's pagination/search/filter/sort
parameters. Catalogs must load the pages needed for selection, rather than assume
the first page is the complete catalog. This API-only change does not modify
`react-admin-kit`; its existing raw-array consumers need a separate migration
before deploying these two versions together.

### Query parameters

- `page`: positive integer, default 1 (one-based).
- `page_size`: integer from 1 to 100, default 20. Invalid/oversized values are
  rejected with 400, never silently clamped.
- `search`: trimmed, case-insensitive substring search, at most 256 Unicode
  characters. `%`, `_` and `!` are literal text, not wildcard operators.
- `sort`: endpoint-specific public key from the server whitelist below.
- `order`: `asc` or `desc`, case-insensitive. Explicit sort without order uses
  ASC. Order without sort changes the configured default ordering's direction.
- Exact/IN filters use one query value: `status=ACTIVE` or
  `status=ACTIVE,INACTIVE`. Empty whole filters are omitted; empty members of a
  non-empty comma-separated list are invalid.

Unknown parameters, repeated keys, malformed URL encoding, invalid directions,
unsupported sorts, invalid filter values, overflowing offsets, or query strings
above 8 KiB receive the existing `{"message":"..."}` 400 response. Search values
and total filter values are bounded (100 filter values, each at most 256 Unicode
characters). Invalid identifier configuration is a developer error, returned as a
sanitized 500 by the existing error layer.

| Endpoint | Searchable fields | Filters | Sort keys | Default order |
| --- | --- | --- | --- | --- |
| `/users` | `full_name`, `username`, `email`, `phone` | `status` (ACTIVE/INACTIVE, normalized uppercase) | `id`, `full_name`, `username`, `email`, `status`, `created_at`, `updated_at`, `last_login_at` | `created_at ASC, id ASC` |
| `/roles` | `name`, `code`, `description` | None | `id`, `name`, `code`, `created_at`, `updated_at` | `code ASC, id ASC` |
| `/permissions` | `name`, `code`, `module`, `description` | `module` (normalized uppercase; at most 100 characters) | `id`, `name`, `code`, `module`, `created_at`, `updated_at` | `module ASC, code ASC, id ASC` |

Every custom ordering appends `id` in the primary direction unless the unique ID
is already included. Database collation determines text ordering; exact/module
filters target stored values. Search applies SQL LOWER on both sides; it does not
promise language-independent Unicode case folding. There is no role filter or
operator DSL without an actual requirement.

Examples:

```text
/users?page=2&page_size=20&search=john&status=ACTIVE&sort=full_name&order=asc
/roles?search=reader&sort=code
/permissions?module=USER,ROLE&page_size=50&sort=code&order=desc
```

## Boundaries and repository use

- `internal/http/request.ParseList` validates HTTP syntax and endpoint parameter
  allowlists. It emits `query.Options`, not SQL.
- `internal/helper/query` defines bounded options and `Page[T]`/`Meta`, with no
  HTTP or database dependencies. Zero-valued internal pagination options receive
  defaults; explicit HTTP zero values are errors. `NewPage` consumes normalized
  options and a non-negative COUNT result.
- Usecases normalize/validate domain values and map entities to existing DTOs.
- `internal/helper/sqlbuilder` resolves trusted definitions and creates an
  immutable plan. `Where()` and `Suffix()` return independent argument slices.
- Repositories retain all SELECT/FROM/COUNT/scanning/assignment SQL and use the
  request context with `GetContext`/`SelectContext`.

A definition uses server-owned column names, optionally qualified by one alias.
Only bare identifiers and `alias.column` are accepted; expressions/fragments are
not allowed. Public sort/filter keys resolve through these maps:

```go
var listDefinition = sqlbuilder.Definition{
    SearchColumns: []string{"u.full_name", "u.email"},
    FilterColumns: map[string]string{"status": "u.status"},
    SortColumns: map[string]string{
        "full_name": "u.full_name",
        "created_at": "u.created_at",
        "id": "u.id",
    },
    DefaultSort: []sqlbuilder.Sort{{Key: "created_at", Order: query.ASC}},
    TieBreaker: "id", // configure a genuinely unique key
}
```

Repository SQL remains visible:

```go
plan, err := sqlbuilder.New(options, listDefinition)
if err != nil {
    return nil, 0, repository.Normalize(err)
}
where, args := plan.Where()
var total int64
if err := r.dbx.GetContext(ctx, &total,
    r.dbx.Rebind(`SELECT COUNT(*) FROM users u`+where), args...); err != nil {
    return nil, 0, repository.Normalize(err)
}
suffix, args := plan.Suffix()
const selectSQL = `SELECT u.id, u.full_name, u.email FROM users u`
if err := r.dbx.SelectContext(ctx, &models,
    r.dbx.Rebind(selectSQL+suffix), args...); err != nil {
    return nil, 0, repository.Normalize(err)
}
```

Rebind the **complete** SQL through the existing sqlx connection. PostgreSQL then
uses `$1...$N`, and MySQL uses `?`. LIMIT/OFFSET and every search/filter value are
parameters. SQL fragments never contain client values or raw client identifiers.
Filter keys are sorted so generated SQL and parameter order are deterministic.
Single text values use `= ?`; multiple values use `IN (?, ...)`. Domain validation
belongs in usecases; complex typed predicates stay as explicit repository SQL.

## Security and performance review

Sort keys only select a whitelisted identifier; direction is checked against
ASC/DESC before SQL assembly. Search/filter columns, aliases, defaults, and the
unique tie-breaker originate entirely in repository code and pass strict
identifier validation. Search escaping also prevents accidental wildcard scans
when the user searches for `%` or `_`. Unit and two-driver integration tests cover
SQL injection attempts, rebinding, count predicates, literal wildcards, paging,
assignment preservation, and cancellation.

Data pagination occurs in SQL, never by slicing a complete result in Go. Count
and page each execute once. User/role assignment loading uses one bounded query
for the current page, replacing the previous query per row. Empty pages skip
assignment queries. Password hashes are not selected for user lists. Count and
page are separate statements; concurrent writes can change their results between
statements. No snapshot transaction is promised.

Current schemas already index status, permission module, unique usernames/email,
and role/permission codes. The default user order can benefit from a compound
`users(created_at, id)` index at scale. Permission default ordering can benefit
from `(module, code, id)`; evaluate plans and expected volume before adding it.
Leading-wildcard, LOWER-based multi-column search generally scans matching table
rows and COUNT may be costly. Ordinary B-tree indexes will not solve arbitrary
substring search. No speculative indexes, full-text framework, or cursor
pagination are added. Very deep OFFSET paging also has the normal database cost.

The older `go-pos-api` pattern informed the explicit SQL/count separation. Its
arbitrary WHERE input, JSON filter/range protocol, loosely typed filter getters,
silent sort fallback, unbounded ranges, shared args, and debug SQL formatting
were not carried over. No implementation was copied.

Unsupported: ORM/model mapping, generated repositories, joins/relationship
builders, client-defined operators or columns, expression languages, nested
condition trees, aggregation/GROUP BY/HAVING/report builders, and cursor paging.
Repositories can bypass the helper entirely for specialized queries.
