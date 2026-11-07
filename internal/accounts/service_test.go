package accounts

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type storeStub struct {
	account Account
	err     error
}

func (stub *storeStub) Ensure(_ context.Context, account Account) error {
	stub.account = account
	return stub.err
}

func TestAccountServiceEnsureAccountCreatesDeterministicAccount(t *testing.T) {
	store := &storeStub{}
	service := NewAccountService(store)

	accountID, err := service.EnsureAccount(context.Background(), "firebase-user")
	if err != nil {
		t.Fatal(err)
	}
	if accountID != DeterministicID("firebase-user") {
		t.Fatal("service did not return the deterministic account ID")
	}
	if store.account.ID != accountID || store.account.FirebaseUID != "firebase-user" {
		t.Fatalf("store received unexpected account: %+v", store.account)
	}
}

func TestAccountServiceEnsureAccountRejectsEmptyUID(t *testing.T) {
	service := NewAccountService(&storeStub{})

	if _, err := service.EnsureAccount(context.Background(), ""); !errors.Is(err, ErrInvalidFirebaseUID) {
		t.Fatalf("expected ErrInvalidFirebaseUID, got %v", err)
	}
}

func TestAccountServiceEnsureAccountReturnsStoreError(t *testing.T) {
	want := errors.New("database unavailable")
	service := NewAccountService(&storeStub{err: want})

	if _, err := service.EnsureAccount(context.Background(), "firebase-user"); !errors.Is(err, want) {
		t.Fatalf("expected store error, got %v", err)
	}
}

func TestDeterministicID(t *testing.T) {
	first := DeterministicID("firebase-user")

	if first == uuid.Nil {
		t.Fatal("deterministic ID must not be nil")
	}
	if first != DeterministicID("firebase-user") {
		t.Fatal("the same Firebase UID produced different account IDs")
	}
	if first == DeterministicID("another-user") {
		t.Fatal("different Firebase UIDs produced the same account ID")
	}
	if DeterministicID("") != uuid.Nil {
		t.Fatal("an empty Firebase UID must produce a nil account ID")
	}
}

func TestUserIDRoundTrip(t *testing.T) {
	want := uuid.New()
	got, ok := UserID(WithUserID(context.Background(), want))
	if !ok || got != want {
		t.Fatalf("unexpected identity: %v %v", got, ok)
	}

	if _, ok := UserID(context.Background()); ok {
		t.Fatal("context without an identity returned an account ID")
	}
	if _, ok := UserID(WithUserID(context.Background(), uuid.Nil)); ok {
		t.Fatal("a nil account ID must not be treated as authenticated")
	}
}
