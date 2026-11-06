package reviews

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BetterGuests/BetterGuests-Server/internal/accounts"
	"github.com/google/uuid"
)

type reviewCreatorStub struct{ err error }

func (stub *reviewCreatorStub) CreateReview(_ context.Context, guestID uuid.UUID, input CreateReviewInput) (Review, error) {
	return Review{ID: uuid.New(), GuestID: guestID, BookingID: input.BookingID}, stub.err
}

func TestCreateReviewHTTPMapsConflict(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", strings.NewReader(`{"booking_id":"11111111-1111-4111-8111-111111111111","rating":5,"comment":"Great"}`))
	request = request.WithContext(accounts.WithUserID(request.Context(), uuid.New()))
	response := httptest.NewRecorder()
	NewHTTPHandler(&reviewCreatorStub{err: ErrReviewAlreadyExists}).CreateReview(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", response.Code)
	}
}

func TestCreateReviewHTTPRequiresAuthentication(t *testing.T) {
	response := httptest.NewRecorder()
	NewHTTPHandler(&reviewCreatorStub{}).CreateReview(response, httptest.NewRequest(http.MethodPost, "/api/v1/reviews", strings.NewReader(`{}`)))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}
