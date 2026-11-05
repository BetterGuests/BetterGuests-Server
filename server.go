package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func createServer() *http.Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &http.Server{
		Addr:              ":" + port,
		Handler:           routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func runServer(server *http.Server) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	done := make(chan error, 1)

	go func() {
		done <- server.ListenAndServe()
	}()

	select {
	case err := <-done:
		return err

	case <-ctx.Done():
		slog.Info("Shutting down...")
	}

	stop()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	if err := <-done; !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	slog.Info("Server stopped gracefully")
	return nil
}
