package main

import (
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
