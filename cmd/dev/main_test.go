package main

import (
	"slices"
	"strings"
	"testing"
)

func TestParseEnv(t *testing.T) {
	values, err := parseEnv(strings.NewReader(`
# comment
PORT=8080
BASE_URL="https://api.example.test/"
export DB_EXPECTED_ROLE='betterguests_app'
`))
	if err != nil {
		t.Fatal(err)
	}
	if values["PORT"] != "8080" {
		t.Fatalf("unexpected port: %q", values["PORT"])
	}
	if values["BASE_URL"] != "https://api.example.test/" {
		t.Fatalf("unexpected base URL: %q", values["BASE_URL"])
	}
	if values["DB_EXPECTED_ROLE"] != "betterguests_app" {
		t.Fatalf("unexpected role: %q", values["DB_EXPECTED_ROLE"])
	}
}

func TestParseEnvRejectsMalformedLine(t *testing.T) {
	if _, err := parseEnv(strings.NewReader("DATABASE_URL")); err == nil {
		t.Fatal("expected malformed environment line to fail")
	}
}

func TestParseEnvSupportsMultilineSingleQuotedValues(t *testing.T) {
	const credentials = `{
  "type": "service_account",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----\n"
}`

	values, err := parseEnv(strings.NewReader(`
PORT=8080
FIREBASE_CREDENTIALS_JSON='` + credentials + `'
BASE_URL="https://api.example.test/"
`))
	if err != nil {
		t.Fatal(err)
	}
	if values["FIREBASE_CREDENTIALS_JSON"] != credentials {
		t.Fatalf("unexpected credentials: %q", values["FIREBASE_CREDENTIALS_JSON"])
	}
	if values["PORT"] != "8080" {
		t.Fatalf("value after the multi-line entry was lost: %q", values["PORT"])
	}
	if values["BASE_URL"] != "https://api.example.test/" {
		t.Fatalf("unexpected base URL: %q", values["BASE_URL"])
	}
}

func TestParseEnvSupportsMultilineDoubleQuotedValues(t *testing.T) {
	values, err := parseEnv(strings.NewReader("MULTILINE=\"line one\nline two\"\nAFTER=ok\n"))
	if err != nil {
		t.Fatal(err)
	}
	if values["MULTILINE"] != "line one\nline two" {
		t.Fatalf("unexpected multi-line value: %q", values["MULTILINE"])
	}
	if values["AFTER"] != "ok" {
		t.Fatalf("value after the multi-line entry was lost: %q", values["AFTER"])
	}
}

func TestParseEnvRejectsUnterminatedQuotedValue(t *testing.T) {
	if _, err := parseEnv(strings.NewReader("MULTILINE='never closed\n")); err == nil {
		t.Fatal("expected an unterminated quoted value to fail")
	}
}

func TestEnvironmentWithOverridesReplacesDatabaseURLs(t *testing.T) {
	environment := environmentWithOverrides([]string{
		"PATH=/bin",
		"DATABASE_URL=postgres://production.example/database",
		"TEST_DATABASE_URL=postgres://production.example/test",
	}, map[string]string{
		"DATABASE_URL":      testDatabaseURL,
		"TEST_DATABASE_URL": testDatabaseURL,
	})

	if !slices.Contains(environment, "DATABASE_URL="+testDatabaseURL) {
		t.Fatal("DATABASE_URL was not overridden with the local test URL")
	}
	if !slices.Contains(environment, "TEST_DATABASE_URL="+testDatabaseURL) {
		t.Fatal("TEST_DATABASE_URL was not overridden with the local test URL")
	}
	for _, entry := range environment {
		if strings.Contains(entry, "production.example") {
			t.Fatalf("production URL leaked into test environment: %q", entry)
		}
	}
}
