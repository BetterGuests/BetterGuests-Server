package reviews

import (
	"context"
	"errors"
	"fmt"

	"github.com/BetterGuests/BetterGuests-Server/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ DB *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{DB: pool} }

func (store *PostgresStore) Create(ctx context.Context, guestID uuid.UUID, review Review) (Review, error) {
	err := db.WithUserTx(ctx, store.DB, guestID.String(), func(tx pgx.Tx) error {
		if err := ensureGuestAccount(ctx, tx, guestID); err != nil {
			return err
		}
		listingID, err := reviewableBookingListing(ctx, tx, review.BookingID, guestID)
		if err != nil {
			return err
		}
		review.ListingID = listingID
		return insertReview(ctx, tx, review)
	})
	if err != nil {
		return Review{}, err
	}
	return review, nil
}

// ensureGuestAccount makes sure a row exists in app.accounts for the guest
// placing the review, since RLS requires every account to exist before it
// can own any data.
func ensureGuestAccount(ctx context.Context, tx pgx.Tx, guestID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `INSERT INTO app.accounts (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, guestID); err != nil {
		return fmt.Errorf("ensure guest account: %w", err)
	}
	return nil
}

// reviewableBookingListing looks up the listing for a booking and confirms
// the guest actually completed it. It returns ErrBookingNotReviewable for
// both "no such booking" and "not completed yet" - callers don't need to
// know which, only that a review isn't allowed.
func reviewableBookingListing(ctx context.Context, tx pgx.Tx, bookingID, guestID uuid.UUID) (uuid.UUID, error) {
	var listingID uuid.UUID
	var status string
	err := tx.QueryRow(ctx, `SELECT listing_id, status FROM app.bookings WHERE id = $1 AND guest_id = $2`, bookingID, guestID).Scan(&listingID, &status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.Nil, ErrBookingNotReviewable
	case err != nil:
		return uuid.Nil, fmt.Errorf("load booking: %w", err)
	case status != "completed":
		return uuid.Nil, ErrBookingNotReviewable
	}
	return listingID, nil
}

// insertReview writes the review row, translating a duplicate booking_id
// (one review per booking, enforced by a unique constraint) into
// ErrReviewAlreadyExists instead of a raw Postgres error.
func insertReview(ctx context.Context, tx pgx.Tx, review Review) error {
	_, err := tx.Exec(ctx, `INSERT INTO app.listing_reviews (id, listing_id, guest_id, booking_id, rating, comment, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, review.ID, review.ListingID, review.GuestID, review.BookingID, review.Rating, review.Comment, review.CreatedAt, review.UpdatedAt)
	var pgError *pgconn.PgError
	switch {
	case errors.As(err, &pgError) && pgError.Code == "23505":
		return ErrReviewAlreadyExists
	case err != nil:
		return fmt.Errorf("insert review: %w", err)
	}
	return nil
}
