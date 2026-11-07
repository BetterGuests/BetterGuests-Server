package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ErrUnauthenticated is returned when a request has no valid bearer token.
var ErrUnauthenticated = errors.New("authentication required")

// AuthService authenticates bearer tokens and resolves them to internal
// accounts.
type AuthService struct {
	verifier    TokenVerifier
	provisioner AccountProvisioner
}

// NewAuthService returns an authentication service backed by verifier and
// provisioner.
func NewAuthService(verifier TokenVerifier, provisioner AccountProvisioner) *AuthService {
	return &AuthService{verifier: verifier, provisioner: provisioner}
}

// Authenticate verifies the Authorization header value and returns the internal
// account it belongs to. Invalid or missing tokens return ErrUnauthenticated;
// provisioning failures return the underlying error so callers can distinguish a
// client problem from a server problem.
func (service *AuthService) Authenticate(ctx context.Context, authorization string) (uuid.UUID, error) {
	rawToken, err := bearerToken(authorization)
	if err != nil {
		return uuid.Nil, ErrUnauthenticated
	}

	token, err := service.verifier.VerifyIDToken(ctx, rawToken)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %v", ErrUnauthenticated, err)
	}
	if token.UID == "" {
		return uuid.Nil, ErrUnauthenticated
	}

	accountID, err := service.provisioner.EnsureAccount(ctx, token.UID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("provision account: %w", err)
	}
	return accountID, nil
}

func bearerToken(authorization string) (string, error) {
	scheme, token, found := strings.Cut(authorization, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", ErrUnauthenticated
	}
	return token, nil
}
