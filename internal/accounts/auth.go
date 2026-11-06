package accounts

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrUnauthenticated = errors.New("authentication required")

type userIDKey struct{}

func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	return userID, ok && userID != uuid.Nil
}

type TokenAuthenticator struct {
	secret []byte
	now    func() time.Time
}

func NewTokenAuthenticator(secret string) (*TokenAuthenticator, error) {
	if len(secret) < 32 {
		return nil, errors.New("AUTH_JWT_SECRET must be at least 32 characters")
	}
	return &TokenAuthenticator{secret: []byte(secret), now: time.Now}, nil
}

func (auth *TokenAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth.authenticate(r.Header.Get("Authorization"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"authentication required"}`))
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), userID)))
	})
}

func (auth *TokenAuthenticator) authenticate(authorization string) (uuid.UUID, error) {
	scheme, token, found := strings.Cut(authorization, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return uuid.Nil, ErrUnauthenticated
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return uuid.Nil, ErrUnauthenticated
	}
	message := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return uuid.Nil, ErrUnauthenticated
	}
	mac := hmac.New(sha256.New, auth.secret)
	_, _ = mac.Write([]byte(message))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return uuid.Nil, ErrUnauthenticated
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "HS256" {
		return uuid.Nil, ErrUnauthenticated
	}
	var claims struct {
		Subject   string `json:"sub"`
		ExpiresAt int64  `json:"exp"`
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(claimsBytes, &claims) != nil || claims.ExpiresAt <= auth.now().Unix() {
		return uuid.Nil, ErrUnauthenticated
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil || userID == uuid.Nil {
		return uuid.Nil, ErrUnauthenticated
	}
	return userID, nil
}
