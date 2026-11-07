package auth

import (
	"encoding/base64"
	"testing"
)

func TestConfigFromLookupRequiresFirebaseSettings(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]string
		wantErr bool
	}{
		{
			name: "valid raw JSON",
			values: map[string]string{
				"FIREBASE_PROJECT_ID":       "betterguests",
				"FIREBASE_CREDENTIALS_JSON": `{"type":"service_account"}`,
			},
		},
		{
			name: "valid multi-line raw JSON",
			values: map[string]string{
				"FIREBASE_PROJECT_ID": "betterguests",
				"FIREBASE_CREDENTIALS_JSON": `{
  "type": "service_account",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----\n"
}`,
			},
		},
		{
			name: "valid base64 JSON",
			values: map[string]string{
				"FIREBASE_PROJECT_ID":       "betterguests",
				"FIREBASE_CREDENTIALS_JSON": base64.StdEncoding.EncodeToString([]byte(`{"type":"service_account"}`)),
			},
		},
		{
			name:    "missing project ID",
			values:  map[string]string{"FIREBASE_CREDENTIALS_JSON": `{}`},
			wantErr: true,
		},
		{
			name:    "missing credentials",
			values:  map[string]string{"FIREBASE_PROJECT_ID": "betterguests"},
			wantErr: true,
		},
		{
			name: "invalid JSON",
			values: map[string]string{
				"FIREBASE_PROJECT_ID":       "betterguests",
				"FIREBASE_CREDENTIALS_JSON": "not-json",
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := configFromLookup(func(key string) string {
				return test.values[key]
			})
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.ProjectID != "betterguests" {
				t.Fatalf("unexpected project ID: %q", cfg.ProjectID)
			}
		})
	}
}
