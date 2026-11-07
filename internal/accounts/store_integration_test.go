package accounts

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/BetterGuests/BetterGuests-Server/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreEnsureProvisionsOnce(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL through go run ./cmd/dev test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	store := NewPostgresAccountStore(pool)
	firebaseUID := "firebase-" + uuid.NewString()
	account := Account{ID: DeterministicID(firebaseUID), FirebaseUID: firebaseUID}

	if err := store.Ensure(ctx, account); err != nil {
		t.Fatal(err)
	}
	if err := store.Ensure(ctx, account); err != nil {
		t.Fatalf("second Ensure was not idempotent: %v", err)
	}

	var storedUID string
	if err := db.WithUserTx(ctx, pool, account.ID.String(), func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT firebase_uid FROM app.accounts WHERE id = $1`, account.ID).Scan(&storedUID)
	}); err != nil {
		t.Fatal(err)
	}
	if storedUID != firebaseUID {
		t.Fatalf("stored firebase UID %q, want %q", storedUID, firebaseUID)
	}

	var visible int
	if err := db.WithUserTx(ctx, pool, uuid.NewString(), func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM app.accounts WHERE id = $1`, account.ID).Scan(&visible)
	}); err != nil {
		t.Fatal(err)
	}
	if visible != 0 {
		t.Fatalf("another account could see the provisioned row: %d", visible)
	}
}
