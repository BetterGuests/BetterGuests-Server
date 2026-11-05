package main

import (
	"log/slog"
	"net/http"
)

func health(w http.ResponseWriter, r *http.Request) {
	slog.Info("Health check requested")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
