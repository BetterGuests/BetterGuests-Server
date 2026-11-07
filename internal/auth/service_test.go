package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type verifierStub struct {
	token FirebaseToken
	err   error
}

func (stub verifierStub) VerifyIDToken(context.Context, string) (FirebaseToken, error) {
	return stub.token, stub.err
}

type provisionerStub struct {
	accountID   uuid.UUID
	firebaseUID string
	err         error
	called      bool
}

func (stub *provisionerStub) EnsureAccount(_ context.Context, firebaseUID string) (uuid.UUID, error) {
	stub.called = true
	stub.firebaseUID = firebaseUID
	if stub.accountID == uuid.Nil {
		stub.accountID = uuid.New()
	}
	return stub.accountID, stub.err
}

func TestAuthServiceAuthenticateReturnsProvisionedAccount(t *testing.T) {
	provisioner := &provisionerStub{}
	service := NewAuthService(verifierStub{token: FirebaseToken{UID: "firebase-user"}}, provisioner)

	accountID, err := service.Authenticate(context.Background(), "Bearer token")
	if err != nil {
		t.Fatal(err)
	}
	if accountID != provisioner.accountID {
		t.Fatalf("unexpected account ID: %v", accountID)
	}
	if !provisioner.called || provisioner.firebaseUID != "firebase-user" {
		t.Fatalf("provisioner not called correctly: %+v", provisioner)
	}
}

func TestAuthServiceAuthenticateRejectsInvalidTokens(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		verifier      verifierStub
	}{
		{name: "missing header"},
		{name: "wrong scheme", authorization: "Basic token"},
		{name: "empty bearer", authorization: "Bearer "},
		{name: "verifier rejects", authorization: "Bearer token", verifier: verifierStub{err: ErrUnauthenticated}},
		{name: "empty provider UID", authorization: "Bearer token", verifier: verifierStub{token: FirebaseToken{}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewAuthService(test.verifier, &provisionerStub{})
			if _, err := service.Authenticate(context.Background(), test.authorization); !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("expected ErrUnauthenticated, got %v", err)
			}
		})
	}
}

func TestAuthServiceAuthenticateReturnsProvisioningError(t *testing.T) {
	want := errors.New("database unavailable")
	service := NewAuthService(
		verifierStub{token: FirebaseToken{UID: "firebase-user"}},
		&provisionerStub{err: want},
	)

	_, err := service.Authenticate(context.Background(), "Bearer token")
	if !errors.Is(err, want) {
		t.Fatalf("expected provisioning error, got %v", err)
	}
	if errors.Is(err, ErrUnauthenticated) {
		t.Fatal("a provisioning failure must not look like an authentication failure")
	}
}
