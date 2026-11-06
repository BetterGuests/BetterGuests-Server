package listings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BetterGuests/BetterGuests-Server/internal/accounts"
	"github.com/google/uuid"
)

type listingCreatorStub struct{ hostID uuid.UUID }

func (stub *listingCreatorStub) CreateListing(_ context.Context, hostID uuid.UUID, _ CreateListingInput) (Listing, error) {
	stub.hostID = hostID
	return Listing{ID: uuid.New(), HostID: hostID}, nil
}

func TestCreateListingHTTPUsesAuthenticatedHost(t *testing.T) {
	creator := &listingCreatorStub{}
	handler := NewHTTPHandler(creator)
	hostID := uuid.New()
	body := `{"title":"Desert House","description":"A quiet and comfortable desert house.","address":{"line1":"1 Main","city":"Phoenix","country_code":"US","latitude":33.4,"longitude":-112.1},"type":"entire_place","property_type":"house","max_guests":4,"minimum_age_to_book":21,"bedrooms":2,"beds":2,"bathrooms":1.5,"amenities":["wifi"]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/listings", strings.NewReader(body))
	request = request.WithContext(accounts.WithUserID(request.Context(), hostID))
	response := httptest.NewRecorder()
	handler.CreateListing(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	if creator.hostID != hostID {
		t.Fatal("handler did not use authenticated host")
	}
}

func TestCreateListingHTTPRequiresAuthentication(t *testing.T) {
	response := httptest.NewRecorder()
	NewHTTPHandler(&listingCreatorStub{}).CreateListing(response, httptest.NewRequest(http.MethodPost, "/api/v1/listings", strings.NewReader(`{}`)))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}
