package migrations

import "embed"

// FS holds the .sql migration files; the runner in internal/postgres reads them.
//
//go:embed *.sql
var FS embed.FS
