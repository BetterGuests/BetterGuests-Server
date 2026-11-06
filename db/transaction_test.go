package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

type panicBeginner struct{}

func (panicBeginner) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	panic("BeginTx must not be called for an invalid user ID")
}

func TestWithUserTxRejectsInvalidUserIDBeforeDatabaseAccess(t *testing.T) {
	err := WithUserTx(context.Background(), panicBeginner{}, "not-a-uuid", func(pgx.Tx) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected an invalid UUID to fail")
	}
}
