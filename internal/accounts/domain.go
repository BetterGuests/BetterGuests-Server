package accounts

import (
	"context"

	"github.com/google/uuid"
)

// T Y P E S --------------------------------------- //

// Account is an internal BetterGuests account linked to a Firebase identity.
type Account struct {
	ID          uuid.UUID
	FirebaseUID string
}

// AccountStore persists accounts.
type AccountStore interface {
	Ensure(ctx context.Context, account Account) error
}
