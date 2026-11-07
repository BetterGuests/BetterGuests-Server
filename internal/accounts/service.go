package accounts

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidFirebaseUID is returned when an empty Firebase UID reaches the
// account mapping.
var ErrInvalidFirebaseUID = errors.New("firebase UID is required")

// firebaseNamespace fixes the UUID namespace used to derive account IDs from
// Firebase UIDs. It must never change: a different namespace would orphan every
// account created before the change.
var firebaseNamespace = uuid.MustParse("3f2a6c1e-1b2d-4e5f-8a9b-0c1d2e3f4a5b")

// AccountService owns the account use cases.
type AccountService struct {
	store AccountStore
}

// NewAccountService returns an account service backed by store.
func NewAccountService(store AccountStore) *AccountService {
	return &AccountService{store: store}
}

// EnsureAccount returns the stable account ID for a verified Firebase UID and
// creates the matching row if it does not exist yet.
func (service *AccountService) EnsureAccount(ctx context.Context, firebaseUID string) (uuid.UUID, error) {
	account := Account{ID: DeterministicID(firebaseUID), FirebaseUID: firebaseUID}
	if account.ID == uuid.Nil {
		return uuid.Nil, ErrInvalidFirebaseUID
	}
	if err := service.store.Ensure(ctx, account); err != nil {
		return uuid.Nil, err
	}
	return account.ID, nil
}

// DeterministicID maps a verified Firebase UID to a stable internal account ID.
// Deriving the ID instead of generating a random one lets a request resolve the
// account with the existing id-based row-level-security policies, without a
// second lookup path keyed on firebase_uid.
func DeterministicID(firebaseUID string) uuid.UUID {
	if firebaseUID == "" {
		return uuid.Nil
	}
	return uuid.NewSHA1(firebaseNamespace, []byte(firebaseUID))
}

type userIDKey struct{}

// WithUserID returns a copy of ctx carrying the authenticated account ID.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID returns the authenticated account ID stored in ctx. The second result
// is false when the request has no authenticated identity.
func UserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	return userID, ok && userID != uuid.Nil
}
