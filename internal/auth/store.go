package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// FirebaseTokenVerifier verifies ID tokens with the Firebase Admin SDK.
type FirebaseTokenVerifier struct {
	client *firebaseauth.Client
}

// ConfigFromEnv builds a FirebaseConfig from environment variables.
func ConfigFromEnv() (FirebaseConfig, error) {
	return configFromLookup(os.Getenv)
}

// configFromLookup does the real work of building a FirebaseConfig, but takes a
// lookup function so tests can supply values without touching the environment.
func configFromLookup(lookup func(string) string) (FirebaseConfig, error) {
	projectID := strings.TrimSpace(lookup("FIREBASE_PROJECT_ID"))
	if projectID == "" {
		return FirebaseConfig{}, errors.New("FIREBASE_PROJECT_ID is required")
	}

	credentials := strings.TrimSpace(lookup("FIREBASE_CREDENTIALS_JSON"))
	if credentials == "" {
		return FirebaseConfig{}, errors.New("FIREBASE_CREDENTIALS_JSON is required")
	}
	credentialsJSON, err := decodeCredentials(credentials)
	if err != nil {
		return FirebaseConfig{}, err
	}

	return FirebaseConfig{ProjectID: projectID, CredentialsJSON: credentialsJSON}, nil
}

// decodeCredentials accepts either raw service-account JSON or the same JSON
// encoded with standard base64. Railway stores multi-line values more reliably
// in the encoded form, so both are supported.
func decodeCredentials(value string) ([]byte, error) {
	if strings.HasPrefix(value, "{") {
		if !json.Valid([]byte(value)) {
			return nil, errors.New("FIREBASE_CREDENTIALS_JSON is not valid JSON")
		}
		return []byte(value), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode FIREBASE_CREDENTIALS_JSON: %w", err)
	}
	if !json.Valid(decoded) {
		return nil, errors.New("FIREBASE_CREDENTIALS_JSON is not valid JSON")
	}
	return decoded, nil
}

// NewFirebaseTokenVerifier builds a verifier from cfg.
func NewFirebaseTokenVerifier(ctx context.Context, cfg FirebaseConfig) (*FirebaseTokenVerifier, error) {
	// WithAuthCredentialsJSON pins the credential type to a service account, so
	// an accidentally supplied configuration (for example the web client
	// config) is rejected instead of being loaded blindly.
	app, err := firebase.NewApp(
		ctx,
		&firebase.Config{ProjectID: cfg.ProjectID},
		option.WithAuthCredentialsJSON(option.ServiceAccount, cfg.CredentialsJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("create firebase app: %w", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("create firebase auth client: %w", err)
	}

	return &FirebaseTokenVerifier{client: client}, nil
}

// VerifyIDToken verifies rawIDToken and returns the Firebase user ID it carries.
func (verifier *FirebaseTokenVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (FirebaseToken, error) {
	token, err := verifier.client.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		return FirebaseToken{}, fmt.Errorf("verify firebase token: %w", err)
	}
	return FirebaseToken{UID: token.UID}, nil
}
