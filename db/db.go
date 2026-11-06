// Package db holds everything related to talking to the PostgreSQL database:
// reading connection settings, opening a connection pool, and running
// queries safely inside per-user transactions.
package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// defaultMaxConns is how many database connections we open if the
	// DB_MAX_CONNS environment variable isn't set.
	defaultMaxConns = int32(5)

	// roleConnLimit is the hard ceiling on connections for the "app" database
	// role. PostgreSQL itself enforces a limit for this role, so we check it
	// here too and fail fast with a clear error instead of a confusing one
	// from the database later.
	roleConnLimit = int32(20)
)

var errInvalidUserID = errors.New("user ID must be a valid UUID")

// Config holds all the settings needed to open a connection pool to the
// database. We only ever build this from environment variables - see
// ConfigFromEnv below.
//
// Note: migration credentials are intentionally NOT part of this struct.
// The running server should only ever be able to read/write data, never
// change the database schema, so we keep that power out of reach here.
type Config struct {
	URL             string        // the Postgres connection string, e.g. "postgres://user:pass@host/db"
	ExpectedRole    string        // the database role we expect to be connecting as (a safety check)
	ApplicationName string        // shows up in Postgres logs/monitoring so we know which app is connected
	MaxConns        int32         // the most connections the pool is allowed to open at once
	MinIdleConns    int32         // how many "warm" idle connections to keep ready
	ConnectTimeout  time.Duration // how long to wait when opening a brand new connection
	PingTimeout     time.Duration // how long to wait when checking a connection is alive
	MaxConnLifetime time.Duration // how long a single connection is allowed to live before it's recycled
	MaxConnIdleTime time.Duration // how long a connection can sit unused before it's closed
	HealthCheck     time.Duration // how often the pool checks on its idle connections
}

// transactionBeginner is anything that can start a transaction - normally
// that's our *pgxpool.Pool. We write it as an interface (a description of
// "what it can do" rather than "what it is") so that tests can pass in a
// fake version that doesn't need a real database at all.
type transactionBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// ConfigFromEnv builds a Config by reading environment variables.
func ConfigFromEnv() (Config, error) {
	return configFromLookup(os.Getenv)
}

// configFromLookup does the real work of building a Config, but takes a
// "lookup" function as an argument so tests can supply fake values instead of
// setting real environment variables.
func configFromLookup(lookup func(string) string) (Config, error) {
	url := lookup("DATABASE_URL")
	if url == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	maxConns, err := maxConnections(lookup("DB_MAX_CONNS"))
	if err != nil {
		return Config{}, err
	}

	expectedRole := lookup("DB_EXPECTED_ROLE")
	if expectedRole == "" {
		expectedRole = "betterguests_app"
	}

	return Config{
		URL:             url,
		ExpectedRole:    expectedRole,
		ApplicationName: "betterguests-api",
		MaxConns:        maxConns,
		MinIdleConns:    1,
		ConnectTimeout:  5 * time.Second,
		PingTimeout:     5 * time.Second,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		HealthCheck:     30 * time.Second,
	}, nil
}

// maxConnections turns DB_MAX_CONNS into a validated int32. An empty string
// means "not set", so we fall back to the default.
func maxConnections(value string) (int32, error) {
	if value == "" {
		return defaultMaxConns, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 || parsed > int(roleConnLimit) {
		return 0, fmt.Errorf("DB_MAX_CONNS must be between 1 and %d", roleConnLimit)
	}
	return int32(parsed), nil
}

// Open creates a database connection pool from a Config and checks that it
// actually works before handing it back to the caller.
//
// A "pool" is just a reusable set of open database connections. Opening a
// new connection for every request would be slow, so instead we open a
// handful up front and hand them out as needed.
func Open(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(config.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinIdleConns = config.MinIdleConns
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.HealthCheck

	poolConfig.ConnConfig.ConnectTimeout = config.ConnectTimeout
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = map[string]string{}
	}
	poolConfig.ConnConfig.RuntimeParams["application_name"] = config.ApplicationName
	poolConfig.AfterRelease = func(connection *pgx.Conn) bool {
		resetCtx, cancel := context.WithTimeout(context.Background(), config.PingTimeout)
		defer cancel()
		_, err := connection.Exec(resetCtx, "RESET ALL")
		return err == nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := verifyPool(ctx, pool, config); err != nil {
		// We opened the pool but it's not safe to use, so close it again
		// instead of leaking connections.
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// verifyPool checks that we can actually talk to the database, and that
// we're connected as the role we expect (e.g. "betterguests_app", not some
// admin role). Asking for current_user already proves the connection is
// alive, so no separate ping is needed.
func verifyPool(ctx context.Context, pool *pgxpool.Pool, config Config) error {
	queryCtx, cancel := context.WithTimeout(ctx, config.PingTimeout)
	defer cancel()

	var role string
	if err := pool.QueryRow(queryCtx, "SELECT current_user::text").Scan(&role); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	if role != config.ExpectedRole {
		return fmt.Errorf("connected as role %q, expected %q", role, config.ExpectedRole)
	}

	return nil
}

// WithUserTx runs fn inside a single database transaction tagged with the
// given user's ID. Use this for every query that touches user-owned data.
//
// Our tables use PostgreSQL Row-Level Security (RLS): the database refuses to
// return rows that don't belong to the current user. To make that work, every
// query tells Postgres "I am acting on behalf of this user" via the
// transaction-local set_config call below. PostgreSQL wipes it clean on
// commit or rollback, so one user's identity can't leak through a pooled
// connection into the next request.
//
// If fn returns an error, we roll back and return that same error. Otherwise
// we commit.
func WithUserTx(
	ctx context.Context,
	db transactionBeginner,
	userID string,
	fn func(pgx.Tx) error,
) error {
	// Fail fast on a malformed ID so it never reaches a query unchecked.
	var parsedID pgtype.UUID
	if err := parsedID.Scan(userID); err != nil || !parsedID.Valid {
		return errInvalidUserID
	}

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin user transaction: %w", err)
	}
	// Rollback unconditionally so an early return never leaves the
	// transaction open. Rolling back after a successful commit is harmless.
	// A fresh context is used because ctx may already be canceled here.
	defer func() { _ = tx.Rollback(context.Background()) }()

	// true means "only for this transaction", so the setting disappears
	// automatically on commit or rollback.
	var configuredID string
	if err := tx.QueryRow(
		ctx,
		"SELECT set_config('app.user_id', $1, true)",
		userID,
	).Scan(&configuredID); err != nil {
		return fmt.Errorf("bind database user: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit user transaction: %w", err)
	}
	return nil
}
