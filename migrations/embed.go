package migrations

import "embed"

// Files makes migrations independent of the process working directory.
//
//go:embed postgres/*.sql mysql/*.sql
var Files embed.FS
