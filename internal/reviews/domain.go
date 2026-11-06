package reviews

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID        uuid.UUID `json:"id"`
	ListingID uuid.UUID `json:"listing_id"`
	BookingID uuid.UUID `json:"booking_id"`
	GuestID   uuid.UUID `json:"guest_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateReviewInput struct {
	BookingID uuid.UUID `json:"booking_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment"`
}
