package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/BetterGuests/BetterGuests-Server/db"
	"github.com/BetterGuests/BetterGuests-Server/internal/accounts"
	"github.com/BetterGuests/BetterGuests-Server/internal/auth"
	"github.com/BetterGuests/BetterGuests-Server/internal/listings"
	"github.com/BetterGuests/BetterGuests-Server/internal/logging"
	"github.com/BetterGuests/BetterGuests-Server/internal/reviews"
)

func main() {
	logging.Configure()

	if err := run(); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {

	// D A T A B A S E

	config, err := db.ConfigFromEnv()
	if err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, config)
	if err != nil {
		return fmt.Errorf("database connection: %w", err)
	}
	defer pool.Close()

	// S E R V E R

	origins, err := allowedOrigins()
	if err != nil {
		return fmt.Errorf("CORS configuration: %w", err)
	}
	firebaseConfig, err := auth.ConfigFromEnv()
	if err != nil {
		return fmt.Errorf("firebase configuration: %w", err)
	}
	firebaseVerifier, err := auth.NewFirebaseTokenVerifier(ctx, firebaseConfig)
	if err != nil {
		return fmt.Errorf("firebase configuration: %w", err)
	}
	accountService := accounts.NewAccountService(accounts.NewPostgresAccountStore(pool))
	authService := auth.NewAuthService(firebaseVerifier, accountService)
	authenticator := auth.NewAuthenticator(authService)
	listingHandler := listings.NewHTTPHandler(listings.NewListingService(listings.NewPostgresListingStore(pool)))
	reviewHandler := reviews.NewHTTPHandler(reviews.NewService(reviews.NewPostgresStore(pool)))
	api := http.NewServeMux()
	api.HandleFunc("POST /api/v1/listings", listingHandler.CreateListing)
	api.HandleFunc("POST /api/v1/reviews", reviewHandler.CreateReview)
	server, err := createServer(routes(pool, origins, authenticator.Middleware(api)))
	if err != nil {
		return fmt.Errorf("server creation: %w", err)
	}

	return runServer(server)
}

func allowedOrigins() ([]string, error) {
	var origins []string
	if err := json.Unmarshal([]byte(os.Getenv("ALLOWED_ORIGINS")), &origins); err != nil {
		return nil, err
	}
	return origins, nil
}
