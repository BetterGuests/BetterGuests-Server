package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/BetterGuests/BetterGuests-Server/internal/accounts"
)

// Authenticator is HTTP middleware that authenticates bearer tokens.
type Authenticator struct {
	service *AuthService
}

// NewAuthenticator returns middleware backed by service.
func NewAuthenticator(service *AuthService) *Authenticator {
	return &Authenticator{service: service}
}

// Middleware verifies the Authorization bearer token, ensures the account row
// exists, and stores the internal account ID on the request context.
func (auth *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID, err := auth.service.Authenticate(r.Context(), r.Header.Get("Authorization"))
		switch {
		case errors.Is(err, ErrUnauthenticated):
			slog.Debug("Authentication rejected", "error", err)
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		case err != nil:
			slog.Error("Authenticate request", "error", err)
			writeError(w, http.StatusInternalServerError, "could not authenticate request")
			return
		}

		next.ServeHTTP(w, r.WithContext(accounts.WithUserID(r.Context(), accountID)))
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
