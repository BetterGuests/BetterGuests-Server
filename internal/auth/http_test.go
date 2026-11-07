package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BetterGuests/BetterGuests-Server/internal/accounts"
	"github.com/google/uuid"
)

func TestMiddlewareStoresAuthenticatedAccount(t *testing.T) {
	provisioner := &provisionerStub{accountID: uuid.New()}
	authenticator := NewAuthenticator(NewAuthService(
		verifierStub{token: FirebaseToken{UID: "firebase-user"}},
		provisioner,
	))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID, ok := accounts.UserID(r.Context())
		if !ok || accountID != provisioner.accountID {
			t.Fatalf("unexpected account: %v %v", accountID, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}
}

func TestMiddlewareRejectsUnauthenticatedRequests(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	authenticator := NewAuthenticator(NewAuthService(verifierStub{}, &provisionerStub{}))
	authenticator.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unauthenticated request reached handler")
	})).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestMiddlewareReturnsServerErrorWhenProvisioningFails(t *testing.T) {
	authenticator := NewAuthenticator(NewAuthService(
		verifierStub{token: FirebaseToken{UID: "firebase-user"}},
		&provisionerStub{err: errors.New("database unavailable")},
	))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	authenticator.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request reached handler despite provisioning failure")
	})).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", response.Code)
	}
}
