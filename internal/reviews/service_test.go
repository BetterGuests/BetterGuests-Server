package reviews

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type reviewStoreStub struct{ err error }

func (stub *reviewStoreStub) Create(_ context.Context, _ uuid.UUID, review Review) (Review, error) {
	return review, stub.err
}

func TestCreateReviewValidatesAndUsesAuthenticatedGuest(t *testing.T) {
	service := NewService(&reviewStoreStub{})
	guestID := uuid.New()
	bookingID := uuid.New()
	review, err := service.CreateReview(context.Background(), guestID, CreateReviewInput{BookingID: bookingID, Rating: 5, Comment: "  Great stay  "})
	if err != nil {
		t.Fatal(err)
	}
	if review.GuestID != guestID || review.BookingID != bookingID || review.Comment != "Great stay" {
		t.Fatalf("unexpected review: %+v", review)
	}
}

func TestCreateReviewRejectsInvalidRating(t *testing.T) {
	_, err := NewService(&reviewStoreStub{}).CreateReview(context.Background(), uuid.New(), CreateReviewInput{BookingID: uuid.New(), Rating: 6, Comment: "No"})
	if !errors.Is(err, ErrInvalidReview) {
		t.Fatalf("expected invalid review, got %v", err)
	}
}

func TestCreateReviewReturnsAuthorizationError(t *testing.T) {
	_, err := NewService(&reviewStoreStub{err: ErrBookingNotReviewable}).CreateReview(context.Background(), uuid.New(), CreateReviewInput{BookingID: uuid.New(), Rating: 4, Comment: "Good"})
	if !errors.Is(err, ErrBookingNotReviewable) {
		t.Fatalf("expected booking error, got %v", err)
	}
}
