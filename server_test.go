package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubPinger struct {
	err error
}

func (p stubPinger) Ping(context.Context) error {
	return p.err
}

func TestHealthIsIndependentOfDatabase(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	handler := routes(
		stubPinger{err: errors.New("database unavailable")},
		[]string{"http://localhost:3000"},
	)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
}

func TestServerHasDefensiveTimeouts(t *testing.T) {
	server, err := createServer(http.NewServeMux())
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	if server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("unexpected read-header timeout: %s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != 15*time.Second {
		t.Fatalf("unexpected read timeout: %s", server.ReadTimeout)
	}
	if server.WriteTimeout != 30*time.Second {
		t.Fatalf("unexpected write timeout: %s", server.WriteTimeout)
	}
	if server.IdleTimeout != 60*time.Second {
		t.Fatalf("unexpected idle timeout: %s", server.IdleTimeout)
	}
	if server.MaxHeaderBytes != 1<<20 {
		t.Fatalf("unexpected maximum header size: %d", server.MaxHeaderBytes)
	}
}

func TestReadinessChecksDatabase(t *testing.T) {
	tests := []struct {
		name       string
		pingError  error
		wantStatus int
	}{
		{name: "ready", wantStatus: http.StatusOK},
		{
			name:       "database unavailable",
			pingError:  errors.New("unavailable"),
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/ready", nil)
			response := httptest.NewRecorder()

			handler := routes(
				stubPinger{err: test.pingError},
				[]string{"http://localhost:3000"},
			)

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf(
					"expected status %d, got %d",
					test.wantStatus,
					response.Code,
				)
			}
		})
	}
}
