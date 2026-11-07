package listings

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

type listingStoreStub struct {
	saved  Listing
	userID string
	err    error
}

func (store *listingStoreStub) Create(_ context.Context, userID string, listing Listing) error {
	store.userID = userID
	store.saved = listing
	return store.err
}

func validCreateInput() CreateListingInput {
	return CreateListingInput{Title: "Desert House", Description: "A quiet and comfortable desert house.", Address: Address{Line1: "1 Main St", City: "Phoenix", CountryCode: "us", Latitude: 33.45, Longitude: -112.07}, Type: EntirePlace, PropertyType: House, MaxGuests: 4, MinimumAgeToBook: 21, Bedrooms: 2, Beds: 2, Bathrooms: 1.5, Amenities: []Amenity{WiFi, KitchenAmenity, WiFi}, Photos: []CreateListingPhotoInput{{ObjectKey: "listing/photo.jpg", Position: 0}}, Rules: []HouseRule{{Title: "No smoking"}}}
}

func TestCreateListing(t *testing.T) {
	store := &listingStoreStub{}
	service := NewListingService(store)
	hostID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	listingID := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	photoID := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	ids := []uuid.UUID{listingID, photoID}
	service.newID = func() uuid.UUID { id := ids[0]; ids = ids[1:]; return id }
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.FixedZone("test", -7*60*60))
	service.now = func() time.Time { return now }
	listing, err := service.CreateListing(context.Background(), hostID, validCreateInput())

	if err != nil {
		t.Fatal(err)
	}
	if listing.ID != listingID || listing.HostID != hostID || listing.Status != Draft {
		t.Fatalf("unexpected listing: %+v", listing)
	}
	if listing.CreatedAt.Location() != time.UTC || !listing.CreatedAt.Equal(now) {
		t.Fatalf("unexpected timestamp: %v", listing.CreatedAt)
	}
	if !reflect.DeepEqual(listing.Amenities, []Amenity{WiFi, KitchenAmenity}) {
		t.Fatalf("unexpected amenities: %#v", listing.Amenities)
	}
	if len(listing.Photos) != 1 || listing.Photos[0].ID != photoID {
		t.Fatalf("unexpected photos: %#v", listing.Photos)
	}
	if store.userID != hostID.String() || store.saved.ID != listing.ID {
		t.Fatal("listing was not persisted as authenticated host")
	}
}

func TestCreateListingRejectsUnknownAmenity(t *testing.T) {
	input := validCreateInput()
	input.Amenities = []Amenity{"client_invented"}
	_, err := NewListingService(&listingStoreStub{}).CreateListing(context.Background(), uuid.New(), input)

	if !errors.Is(err, ErrInvalidListing) {
		t.Fatalf("expected invalid listing, got %v", err)
	}
}

func TestCreateListingReturnsStoreError(t *testing.T) {
	want := errors.New("database failed")
	_, err := NewListingService(&listingStoreStub{err: want}).CreateListing(context.Background(), uuid.New(), validCreateInput())

	if !errors.Is(err, want) {
		t.Fatalf("expected store error, got %v", err)
	}
}

// WE need all the other tests and functions here for listin.
