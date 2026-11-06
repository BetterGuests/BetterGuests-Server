// This is a small command-line program for managing database migrations -
// the versioned SQL scripts that create and change our tables over time.
//
// Run it with one of:
//
//	go run ./cmd/migrate up       # apply any migrations that haven't run yet
//	go run ./cmd/migrate status   # show which migrations have run
//	go run ./cmd/migrate version  # print the current migration version number
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/BetterGuests/BetterGuests-Server/db/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var (
	errUsage              = errors.New("usage: migrate <up|status|version>")
	errMissingDatabaseURL = errors.New("MIGRATION_DATABASE_URL is required")
)

func main() {
	if err := run(); err != nil {
		slog.Error("Migration command failed", "error", err)
		os.Exit(1)
	}
}

// run contains the actual logic. Keeping it separate from main lets it
// return a normal Go error, instead of main having to call os.Exit from
// several different places.
func run() error {
	// os.Args is the list of command-line arguments. os.Args[0] is always
	// the program name itself, so os.Args[1] is the first real argument -
	// e.g. "up" in `go run ./cmd/migrate up`.
	if len(os.Args) != 2 {
		return errUsage
	}
	command := os.Args[1]
	if command != "up" && command != "status" && command != "version" {
		return fmt.Errorf("unsupported migration command %q", command)
	}

	databaseURL := os.Getenv("MIGRATION_DATABASE_URL")
	if databaseURL == "" {
		return errMissingDatabaseURL
	}

	// sql.Open doesn't actually connect yet - it just prepares a handle we
	// can use to connect later. "pgx" here is the Postgres driver we
	// imported above for its side effects (the `_ "..."` import).
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return errors.New("MIGRATION_DATABASE_URL is invalid")
	}
	defer database.Close()

	// signal.NotifyContext gives us a context that automatically gets
	// canceled if the user hits Ctrl+C (os.Interrupt) or the process is
	// asked to stop (SIGTERM, e.g. by a deploy tool). Passing ctx into the
	// goose calls below means a migration in progress can be interrupted
	// cleanly instead of being killed mid-write.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := database.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to migration database: %w", err)
	}

	// Safety check: migrations change the shape of the database (tables,
	// columns, permissions), so they must only ever run as the dedicated
	// migrator role - never as the role the live server normally uses.
	// This stops a misconfigured MIGRATION_DATABASE_URL from accidentally
	// running schema changes with the wrong (and more limited) permissions.
	var role string
	if err := database.QueryRowContext(ctx, "SELECT current_user::text").Scan(&role); err != nil {
		return fmt.Errorf("verify migration role: %w", err)
	}
	if role != "betterguests_migrator" {
		return fmt.Errorf("refusing migrations: expected role %q, got %q", "betterguests_migrator", role)
	}

	// goose is the migration library doing the real work. We point it at
	// our embedded SQL files and tell it where to keep its own bookkeeping
	// table (which migrations have already run).
	goose.SetBaseFS(migrations.Files)
	goose.SetTableName("app.goose_db_version")
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("configure migrations: %w", err)
	}

	switch command {
	case "up":
		return goose.UpContext(ctx, database, ".")
	case "status":
		return goose.StatusContext(ctx, database, ".")
	case "version":
		version, err := goose.GetDBVersionContext(ctx, database)
		if err != nil {
			return err
		}
		fmt.Println(version)
		return nil
	default:
		// We already validated command above, so this can never actually
		// run - it's just here so the compiler (and future readers) can
		// see that every case is handled.
		panic("validated command was not handled")
	}
}
