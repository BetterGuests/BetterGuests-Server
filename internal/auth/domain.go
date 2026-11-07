package auth

import (
	"context"

	"github.com/google/uuid"
)

// T Y P E S --------------------------------------- //

// FirebaseToken is the verified identity that the API needs from Firebase.
type FirebaseToken struct {
	UID string
}

// FirebaseConfig holds the settings needed to verify Firebase ID tokens.
type FirebaseConfig struct {
	ProjectID       string
	CredentialsJSON []byte
}

// TokenVerifier verifies an identity-provider token. Handlers and tests depend
// on this interface so they never call the provider directly.
type TokenVerifier interface {
	VerifyIDToken(ctx context.Context, rawIDToken string) (FirebaseToken, error)
}

// AccountProvisioner makes sure an internal account exists for a verified
// identity before a handler uses it.
type AccountProvisioner interface {
	EnsureAccount(ctx context.Context, firebaseUID string) (uuid.UUID, error)
}
