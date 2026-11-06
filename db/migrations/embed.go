package migrations

import "embed"

// Files contains every *.sql file in this folder, baked directly into the
// compiled program at build time.
//
// The //go:embed comment right above the variable isn't a regular comment -
// it's a special instruction the Go compiler reads. It means we don't have
// to ship .sql files alongside the binary or worry about the right folder
// being present on a server; the migration SQL travels inside the program
// itself.
//
//go:embed *.sql
var Files embed.FS
