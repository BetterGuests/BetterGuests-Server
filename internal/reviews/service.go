package reviews

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidReview        = errors.New("invalid review")
	ErrBookingNotReviewable = errors.New("booking is not reviewable")
	ErrReviewAlreadyExists  = errors.New("booking already has a review")
)

type Store interface {
	Create(context.Context, uuid.UUID, Review) (Review, error)
}

type Service struct {
	store Store
	now   func() time.Time
	newID func() uuid.UUID
}

func NewService(store Store) *Service { return &Service{store: store, now: time.Now, newID: uuid.New} }

func (service *Service) CreateReview(ctx context.Context, guestID uuid.UUID, input CreateReviewInput) (Review, error) {
	input.Comment = strings.TrimSpace(input.Comment)
	if input.BookingID == uuid.Nil {
		return Review{}, fmt.Errorf("%w: booking_id is required", ErrInvalidReview)
	}
	if input.Rating < 1 || input.Rating > 5 {
		return Review{}, fmt.Errorf("%w: rating must be between 1 and 5", ErrInvalidReview)
	}
	if len(input.Comment) < 1 || len(input.Comment) > 4000 {
		return Review{}, fmt.Errorf("%w: comment must be 1-4000 characters", ErrInvalidReview)
	}
	now := service.now().UTC()
	return service.store.Create(ctx, guestID, Review{ID: service.newID(), BookingID: input.BookingID, GuestID: guestID, Rating: input.Rating, Comment: input.Comment, CreatedAt: now, UpdatedAt: now})
}
