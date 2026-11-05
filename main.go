package main

import (
	"log/slog"
	"os"
)

func main() {
	server := createServer()

	slog.Info("Starting server", "address", server.Addr)

	if err := runServer(server); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
