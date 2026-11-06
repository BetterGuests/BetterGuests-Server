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
		if _, err := tx.Exec(ctx, `INSERT INTO app.accounts (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, guestID); err != nil {
			return fmt.Errorf("ensure guest account: %w", err)
		}
		var status string
		if err := tx.QueryRow(ctx, `SELECT listing_id, status FROM app.bookings WHERE id = $1 AND guest_id = $2`, review.BookingID, guestID).Scan(&review.ListingID, &status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrBookingNotReviewable
			}
			return fmt.Errorf("load booking: %w", err)
		}
		if status != "completed" {
			return ErrBookingNotReviewable
		}
		_, err := tx.Exec(ctx, `INSERT INTO app.listing_reviews (id, listing_id, guest_id, booking_id, rating, comment, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, review.ID, review.ListingID, review.GuestID, review.BookingID, review.Rating, review.Comment, review.CreatedAt, review.UpdatedAt)
		if err != nil {
			var pgError *pgconn.PgError
			if errors.As(err, &pgError) && pgError.Code == "23505" {
				return ErrReviewAlreadyExists
			}
			return fmt.Errorf("insert review: %w", err)
		}
		return nil
	})
	if err != nil {
		return Review{}, err
	}
	return review, nil
}
