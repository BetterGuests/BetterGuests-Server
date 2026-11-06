package accounts

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAuthenticationMiddlewareValidatesSignedSubject(t *testing.T) {
	secret := "01234567890123456789012345678901"
	authenticator, err := NewTokenAuthenticator(secret)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.New()
	token := signedToken(secret, userID.String(), time.Now().Add(time.Hour).Unix())
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := UserID(r.Context())
		if !ok || got != userID {
			t.Fatalf("unexpected user: %v %v", got, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}
}

func TestAuthenticationMiddlewareRejectsExpiredToken(t *testing.T) {
	secret := "01234567890123456789012345678901"
	authenticator, _ := NewTokenAuthenticator(secret)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+signedToken(secret, uuid.New().String(), time.Now().Add(-time.Minute).Unix()))
	response := httptest.NewRecorder()
	authenticator.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("expired token reached handler") })).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func signedToken(secret, subject string, expiresAt int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + subject + `","exp":` + strconv.FormatInt(expiresAt, 10) + `}`))
	message := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	return message + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
