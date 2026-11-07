package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	attackerUserID = "11111111-1111-4111-8111-111111111111"
	victimUserID   = "22222222-2222-4222-8222-222222222222"
)

func TestDatabaseSecurityBoundary(t *testing.T) {
	appURL := os.Getenv("TEST_DATABASE_URL")
	migrationURL := os.Getenv("TEST_MIGRATION_DATABASE_URL")
	if appURL == "" || migrationURL == "" {
		t.Skip("set TEST_DATABASE_URL and TEST_MIGRATION_DATABASE_URL to run database attack tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	migrationPool := openTestPool(t, ctx, migrationURL, "betterguests_migrator", 2)
	defer migrationPool.Close()
	appPool := openTestPool(t, ctx, appURL, "betterguests_app", 1)
	defer appPool.Close()

	createAttackFixture(t, ctx, migrationPool)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = migrationPool.Exec(cleanupCtx, "DROP TABLE IF EXISTS app.security_attack_accounts")
	})

	insertAccount(t, ctx, appPool, attackerUserID, "attacker@example.test", "attacker-secret")
	insertAccount(t, ctx, appPool, victimUserID, "victim@example.test", "victim-secret")

	t.Run("missing identity is default deny", func(t *testing.T) {
		var count int
		if err := appPool.QueryRow(ctx, "SELECT count(*) FROM app.security_attack_accounts").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("expected no visible rows without identity, got %d", count)
		}
	})

	t.Run("cross tenant read is denied", func(t *testing.T) {
		err := WithUserTx(ctx, appPool, attackerUserID, func(tx pgx.Tx) error {
			var secret string
			err := tx.QueryRow(
				ctx,
				"SELECT secret FROM app.security_attack_accounts WHERE owner_id = $1",
				victimUserID,
			).Scan(&secret)
			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("expected victim row to be invisible, got %v", err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("cross tenant update and delete affect no rows", func(t *testing.T) {
		err := WithUserTx(ctx, appPool, attackerUserID, func(tx pgx.Tx) error {
			updated, err := tx.Exec(
				ctx,
				"UPDATE app.security_attack_accounts SET secret = $1 WHERE owner_id = $2",
				"stolen",
				victimUserID,
			)
			if err != nil {
				return err
			}
			if updated.RowsAffected() != 0 {
				return fmt.Errorf("updated %d victim rows", updated.RowsAffected())
			}

			deleted, err := tx.Exec(
				ctx,
				"DELETE FROM app.security_attack_accounts WHERE owner_id = $1",
				victimUserID,
			)
			if err != nil {
				return err
			}
			if deleted.RowsAffected() != 0 {
				return fmt.Errorf("deleted %d victim rows", deleted.RowsAffected())
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("identity spoofed insert is rejected", func(t *testing.T) {
		err := WithUserTx(ctx, appPool, attackerUserID, func(tx pgx.Tx) error {
			_, err := tx.Exec(
				ctx,
				`INSERT INTO app.security_attack_accounts (id, owner_id, email, secret)
				 VALUES ('33333333-3333-4333-8333-333333333333', $1, 'spoof@example.test', 'spoof')`,
				victimUserID,
			)
			return err
		})
		if err == nil {
			t.Fatal("spoofed insert unexpectedly succeeded")
		}
	})

	t.Run("SQL injection payload remains data", func(t *testing.T) {
		payloads := []string{
			"' OR true; --",
			"' UNION SELECT secret FROM app.security_attack_accounts --",
			"'; DROP TABLE app.security_attack_accounts; --",
			"victim@example.test'/*",
		}
		for _, payload := range payloads {
			err := WithUserTx(ctx, appPool, attackerUserID, func(tx pgx.Tx) error {
				var count int
				if err := tx.QueryRow(
					ctx,
					"SELECT count(*) FROM app.security_attack_accounts WHERE email = $1",
					payload,
				).Scan(&count); err != nil {
					return err
				}
				if count != 0 {
					return fmt.Errorf("injection payload matched %d rows", count)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("payload %q: %v", payload, err)
			}
		}
	})

	t.Run("row security cannot be disabled by app role", func(t *testing.T) {
		err := WithUserTx(ctx, appPool, attackerUserID, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "SET LOCAL row_security = off"); err != nil {
				return err
			}
			var count int
			return tx.QueryRow(
				ctx,
				"SELECT count(*) FROM app.security_attack_accounts",
			).Scan(&count)
		})
		if err == nil {
			t.Fatal("application role bypassed RLS with row_security=off")
		}
	})

	t.Run("role escalation and DDL are rejected", func(t *testing.T) {
		attacks := []string{
			"SET ROLE betterguests_migrator",
			"CREATE TABLE app.unauthorized_table (id integer)",
			"ALTER TABLE app.security_attack_accounts DISABLE ROW LEVEL SECURITY",
			"TRUNCATE app.security_attack_accounts",
			"SELECT * FROM app.goose_db_version",
		}
		for _, attack := range attacks {
			if _, err := appPool.Exec(ctx, attack); err == nil {
				t.Errorf("attack unexpectedly succeeded: %s", attack)
			}
		}
	})

	t.Run("pooled session identity is scrubbed", func(t *testing.T) {
		tamperPooledConnection(t, ctx, appPool, attackerUserID)

		var count int
		if err := appPool.QueryRow(ctx, "SELECT count(*) FROM app.security_attack_accounts").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("session identity leaked through pool and exposed %d rows", count)
		}

		var statementTimeout string
		if err := appPool.QueryRow(ctx, "SHOW statement_timeout").Scan(&statementTimeout); err != nil {
			t.Fatal(err)
		}
		if statementTimeout != "5s" {
			t.Fatalf("session timeout tampering leaked through pool: %q", statementTimeout)
		}
	})

	t.Run("role and catalog invariants hold", func(t *testing.T) {
		assertSecurityCatalog(t, ctx, migrationPool)
	})
}

func openTestPool(
	t *testing.T,
	ctx context.Context,
	url string,
	role string,
	maxConns int32,
) *pgxpool.Pool {
	t.Helper()
	pool, err := Open(ctx, Config{
		URL:             url,
		ExpectedRole:    role,
		ApplicationName: "betterguests-security-test",
		MaxConns:        maxConns,
		MinIdleConns:    0,
		ConnectTimeout:  5 * time.Second,
		PingTimeout:     5 * time.Second,
		MaxConnLifetime: 10 * time.Minute,
		MaxConnIdleTime: time.Minute,
		HealthCheck:     30 * time.Second,
	})
	if err != nil {
		t.Fatalf("open %s test pool: %v", role, err)
	}
	return pool
}

func createAttackFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		DROP TABLE IF EXISTS app.security_attack_accounts;

		CREATE TABLE app.security_attack_accounts (
			id uuid PRIMARY KEY,
			owner_id uuid NOT NULL,
			email text NOT NULL UNIQUE CHECK (length(email) <= 320),
			secret text NOT NULL CHECK (length(secret) <= 1000)
		);

		CREATE INDEX security_attack_accounts_owner_idx
			ON app.security_attack_accounts (owner_id);

		ALTER TABLE app.security_attack_accounts ENABLE ROW LEVEL SECURITY;
		ALTER TABLE app.security_attack_accounts FORCE ROW LEVEL SECURITY;

		CREATE POLICY security_accounts_select ON app.security_attack_accounts
			FOR SELECT TO betterguests_app
			USING (owner_id = app.current_user_id());
		CREATE POLICY security_accounts_insert ON app.security_attack_accounts
			FOR INSERT TO betterguests_app
			WITH CHECK (owner_id = app.current_user_id());
		CREATE POLICY security_accounts_update ON app.security_attack_accounts
			FOR UPDATE TO betterguests_app
			USING (owner_id = app.current_user_id())
			WITH CHECK (owner_id = app.current_user_id());
		CREATE POLICY security_accounts_delete ON app.security_attack_accounts
			FOR DELETE TO betterguests_app
			USING (owner_id = app.current_user_id());

		GRANT SELECT, INSERT, UPDATE, DELETE
			ON app.security_attack_accounts TO betterguests_app;
	`)
	if err != nil {
		t.Fatalf("create attack fixture: %v", err)
	}
}

func insertAccount(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	userID string,
	email string,
	secret string,
) {
	t.Helper()
	err := WithUserTx(ctx, pool, userID, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx,
			`INSERT INTO app.security_attack_accounts (id, owner_id, email, secret)
			 VALUES ($1, $1, $2, $3)`,
			userID,
			email,
			secret,
		)
		return err
	})
	if err != nil {
		t.Fatalf("insert test account %s: %v", email, err)
	}
}

// tamperPooledConnection sets a fake user identity and disables the
// statement timeout on a single pooled connection, then releases it back to
// the pool - simulating a connection that was "contaminated" by a previous
// request, so the test below can check the pool scrubs it clean on reuse.
func tamperPooledConnection(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID string) {
	t.Helper()
	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Release()

	if _, err := connection.Exec(ctx, "SELECT set_config('app.user_id', $1, false)", userID); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, "SET statement_timeout = 0"); err != nil {
		t.Fatal(err)
	}
}

func assertSecurityCatalog(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	for _, role := range []string{"betterguests_app", "betterguests_migrator"} {
		var superuser, createDB, createRole, replication, bypassRLS bool
		err := pool.QueryRow(
			ctx,
			`SELECT rolsuper, rolcreatedb, rolcreaterole, rolreplication, rolbypassrls
			 FROM pg_roles WHERE rolname = $1`,
			role,
		).Scan(&superuser, &createDB, &createRole, &replication, &bypassRLS)
		if err != nil {
			t.Fatalf("inspect role %s: %v", role, err)
		}
		if superuser || createDB || createRole || replication || bypassRLS {
			t.Errorf("role %s has an unsafe role attribute", role)
		}
	}

	var appCanCreate bool
	if err := pool.QueryRow(
		ctx,
		"SELECT has_schema_privilege('betterguests_app', 'app', 'CREATE')",
	).Scan(&appCanCreate); err != nil {
		t.Fatal(err)
	}
	if appCanCreate {
		t.Error("application role can create objects in app schema")
	}

	var appOwned, missingRLS, missingForcedRLS, missingPolicies, publicGrants int
	queries := []struct {
		name   string
		query  string
		target *int
	}{
		{
			name: "app-owned tables",
			query: `SELECT count(*) FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				JOIN pg_roles r ON r.oid = c.relowner
				WHERE n.nspname = 'app' AND c.relkind IN ('r', 'p')
				  AND r.rolname = 'betterguests_app'`,
			target: &appOwned,
		},
		{
			name: "tables without RLS",
			query: `SELECT count(*) FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = 'app' AND c.relkind IN ('r', 'p')
				  AND c.relname <> 'goose_db_version' AND NOT c.relrowsecurity`,
			target: &missingRLS,
		},
		{
			name: "tables without forced RLS",
			query: `SELECT count(*) FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = 'app' AND c.relkind IN ('r', 'p')
				  AND c.relname <> 'goose_db_version' AND NOT c.relforcerowsecurity`,
			target: &missingForcedRLS,
		},
		{
			name: "tables without app policies",
			query: `SELECT count(*) FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = 'app' AND c.relkind IN ('r', 'p')
				  AND c.relname <> 'goose_db_version'
				  AND NOT EXISTS (
				      SELECT 1 FROM pg_policy p
				      WHERE p.polrelid = c.oid
				        AND 'betterguests_app' = ANY (
				            SELECT rolname FROM pg_roles WHERE oid = ANY (p.polroles)
				        )
				  )`,
			target: &missingPolicies,
		},
		{
			name: "PUBLIC table grants",
			query: `SELECT count(*) FROM information_schema.table_privileges
				WHERE table_schema = 'app' AND grantee = 'PUBLIC'`,
			target: &publicGrants,
		},
	}

	for _, check := range queries {
		if err := pool.QueryRow(ctx, check.query).Scan(check.target); err != nil {
			t.Fatalf("inspect %s: %v", check.name, err)
		}
		if *check.target != 0 {
			t.Errorf("found %d %s", *check.target, check.name)
		}
	}
}
