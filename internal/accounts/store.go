package accounts

import (
	"context"
	"fmt"

	"github.com/BetterGuests/BetterGuests-Server/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAccountStore provisions rows in app.accounts.
type PostgresAccountStore struct{ DB *pgxpool.Pool }

// NewPostgresAccountStore returns an account store backed by pool.
func NewPostgresAccountStore(pool *pgxpool.Pool) *PostgresAccountStore {
	return &PostgresAccountStore{DB: pool}
}

// Ensure creates the account row for an already-verified identity. It is
// idempotent because every authenticated request calls it before a handler runs.
func (store *PostgresAccountStore) Ensure(ctx context.Context, account Account) error {
	return db.WithUserTx(ctx, store.DB, account.ID.String(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO app.accounts (id, firebase_uid) VALUES ($1, NULLIF($2, '')) ON CONFLICT (id) DO NOTHING`,
			account.ID,
			account.FirebaseUID,
		); err != nil {
			return fmt.Errorf("ensure account: %w", err)
		}
		return nil
	})
}
