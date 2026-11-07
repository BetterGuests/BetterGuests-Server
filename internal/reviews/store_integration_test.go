package reviews

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/BetterGuests/BetterGuests-Server/db"
	"github.com/BetterGuests/BetterGuests-Server/internal/listings"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReviewStoreRequiresCompletedGuestBooking(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL through go run ./cmd/dev test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	hostID, guestID, otherGuestID := uuid.New(), uuid.New(), uuid.New()
	listing, err := listings.NewListingService(listings.NewPostgresListingStore(pool)).CreateListing(ctx, hostID, listings.CreateListingInput{
		Title: "Integration House", Description: "A complete listing used by the integration test.",
		Address: listings.Address{Line1: "1 Test Way", City: "Phoenix", CountryCode: "US", Latitude: 33.4, Longitude: -112.1},
		Type:    listings.EntirePlace, PropertyType: listings.House, MaxGuests: 4,
		MinimumAgeToBook: 21, Bedrooms: 2, Beds: 2, Bathrooms: 1.5,
		Amenities: []listings.Amenity{listings.WiFi, listings.KitchenAmenity},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := publishListing(ctx, pool, hostID, listing.ID); err != nil {
		t.Fatal(err)
	}
	bookingID := uuid.New()
	if err := createPendingBooking(ctx, pool, guestID, listing.ID, bookingID); err != nil {
		t.Fatal(err)
	}

	// Each subtest below checks exactly one policy decision. Keeping them
	// separate (rather than one function full of unrelated if-checks) means
	// a failure names the behavior that broke instead of just a line number.
	t.Run("guest cannot complete their own booking", func(t *testing.T) {
		if err := completeBooking(ctx, pool, guestID, bookingID, 0); err != nil {
			t.Fatalf("guest was allowed to complete their own booking: %v", err)
		}
	})

	t.Run("host can complete the booking", func(t *testing.T) {
		if err := completeBooking(ctx, pool, hostID, bookingID, 1); err != nil {
			t.Fatalf("host could not complete booking: %v", err)
		}
	})

	service := NewService(NewPostgresStore(pool))

	t.Run("guest can review their completed booking", func(t *testing.T) {
		created, err := service.CreateReview(ctx, guestID, CreateReviewInput{BookingID: bookingID, Rating: 5, Comment: "Excellent stay"})
		if err != nil {
			t.Fatal(err)
		}
		if created.ListingID != listing.ID || created.GuestID != guestID {
			t.Fatalf("review used wrong ownership: %+v", created)
		}
	})

	t.Run("the same booking cannot be reviewed twice", func(t *testing.T) {
		if _, err := service.CreateReview(ctx, guestID, CreateReviewInput{BookingID: bookingID, Rating: 4, Comment: "Again"}); !errors.Is(err, ErrReviewAlreadyExists) {
			t.Fatalf("expected duplicate review error, got %v", err)
		}
	})

	t.Run("only the booking's own guest can review it", func(t *testing.T) {
		if _, err := service.CreateReview(ctx, otherGuestID, CreateReviewInput{BookingID: bookingID, Rating: 4, Comment: "Spoofed"}); !errors.Is(err, ErrBookingNotReviewable) {
			t.Fatalf("expected unauthorized booking error, got %v", err)
		}
	})
}

func publishListing(ctx context.Context, pool *pgxpool.Pool, hostID, listingID uuid.UUID) error {
	return db.WithUserTx(ctx, pool, hostID.String(), func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE app.listings SET status = 'published' WHERE id = $1`, listingID)
		return err
	})
}

func createPendingBooking(ctx context.Context, pool *pgxpool.Pool, guestID, listingID, bookingID uuid.UUID) error {
	return db.WithUserTx(ctx, pool, guestID.String(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO app.accounts (id) VALUES ($1)`, guestID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO app.bookings (id, listing_id, guest_id, status, starts_on, ends_on) VALUES ($1,$2,$3,'pending',CURRENT_DATE - 2,CURRENT_DATE - 1)`, bookingID, listingID, guestID)
		return err
	})
}

// completeBooking marks bookingID completed as actingUserID and fails unless
// exactly wantRowsAffected rows were changed - RLS silently affects zero rows
// instead of erroring, so the row count is the only way to tell "blocked by
// policy" apart from "succeeded".
func completeBooking(ctx context.Context, pool *pgxpool.Pool, actingUserID, bookingID uuid.UUID, wantRowsAffected int64) error {
	return db.WithUserTx(ctx, pool, actingUserID.String(), func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, `UPDATE app.bookings SET status = 'completed', completed_at = now() WHERE id = $1`, bookingID)
		if err != nil {
			return err
		}
		if result.RowsAffected() != wantRowsAffected {
			return fmt.Errorf("expected %d row(s) affected, got %d", wantRowsAffected, result.RowsAffected())
		}
		return nil
	})
}
