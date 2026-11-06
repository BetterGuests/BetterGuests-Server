package listings

import (
	"context"
	"fmt"

	"github.com/BetterGuests/BetterGuests-Server/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ListingStore interface {
	Create(
		ctx context.Context,
		userID string,
		listing Listing,
	) error
}

type PostgresListingStore struct {
	DB *pgxpool.Pool
}

func NewPostgresListingStore(pool *pgxpool.Pool) *PostgresListingStore {
	return &PostgresListingStore{DB: pool}
}

func (s *PostgresListingStore) Create(
	ctx context.Context,
	userID string,
	listing Listing,
) error {

	return db.WithUserTx(ctx, s.DB, userID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO app.accounts (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, listing.HostID); err != nil {
			return fmt.Errorf("ensure host account: %w", err)
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO app.listings (
				id,
				host_id,
				title,
				description,
				address_line1,
				address_line2,
				city,
				region,
				postal_code,
				country_code,
				latitude,
				longitude,
				place_id,
				listing_type,
				property_type,
				max_guests,
				minimum_age_to_book,
				bedrooms,
				beds,
				bathrooms,
				status,
				created_at,
				updated_at
			) VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				NULLIF($6,''),
				$7,
				NULLIF($8,''),
				NULLIF($9,''),
				$10,
				$11,
				$12,
				NULLIF($13,''),
				$14,
				$15,
				$16,
				$17,
				$18,
				$19,
				$20,
				$21,
				$22,
				$23
			)
        `,
			listing.ID,
			listing.HostID,
			listing.Title,
			listing.Description,
			listing.Address.Line1,
			listing.Address.Line2,
			listing.Address.City,
			listing.Address.Region,
			listing.Address.PostalCode,
			listing.Address.CountryCode,
			listing.Address.Latitude,
			listing.Address.Longitude,
			listing.Address.PlaceID,
			listing.Type,
			listing.PropertyType,
			listing.MaxGuests,
			listing.MinimumAgeToBook,
			listing.Bedrooms,
			listing.Beds,
			listing.Bathrooms,
			listing.Status,
			listing.CreatedAt,
			listing.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert listing: %w", err)
		}
		for _, amenity := range listing.Amenities {
			result, err := tx.Exec(ctx, `
				INSERT INTO app.listing_amenities (id, listing_id, amenity_key)
				SELECT $1, $2, key FROM app.amenities WHERE key = $3
			`, uuid.New(), listing.ID, amenity)
			if err != nil {
				return fmt.Errorf("link amenity %q: %w", amenity, err)
			}
			if result.RowsAffected() != 1 {
				return fmt.Errorf("amenity catalog is missing %q", amenity)
			}
		}
		for _, photo := range listing.Photos {
			if _, err := tx.Exec(ctx, `INSERT INTO app.listing_photos (id, listing_id, object_key, alt_text, position) VALUES ($1,$2,$3,NULLIF($4,''),$5)`, photo.ID, listing.ID, photo.ObjectKey, photo.AltText, photo.Position); err != nil {
				return fmt.Errorf("insert listing photo: %w", err)
			}
		}
		for position, rule := range listing.Rules {
			if _, err := tx.Exec(ctx, `INSERT INTO app.listing_rules (id, listing_id, title, description, position) VALUES ($1,$2,$3,NULLIF($4,''),$5)`, uuid.New(), listing.ID, rule.Title, rule.Description, position); err != nil {
				return fmt.Errorf("insert listing rule: %w", err)
			}
		}
		return nil
	})
}
