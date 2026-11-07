package listings

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ListingService owns the listing use cases. ListingStore is the boundary it uses
// to persist domain objects; the clock and ID generator make creation
// deterministic in tests.
type ListingService struct {
	store ListingStore
	now   func() time.Time
	newID func() uuid.UUID
}

func NewListingService(store ListingStore) *ListingService {
	return &ListingService{
		store: store,
		now:   time.Now,
		newID: uuid.New,
	}
}

// CreateListingInput contains only values a host may provide when creating a
// listing. HostID comes from authentication; ID, status, and timestamps are
// assigned by ListingService.
type CreateListingInput struct {
	Title            string
	Description      string
	Address          Address
	Type             ListingType
	PropertyType     PropertyType
	MaxGuests        int
	MinimumAgeToBook int
	Bedrooms         int
	Beds             int
	Bathrooms        float64
	Amenities        []Amenity
	Photos           []CreateListingPhotoInput
	Rules            []HouseRule
}

// CreateListingPhotoInput references a photo that has already been uploaded
// to object storage. A presigned read URL is generated later and is never
// accepted as permanent listing data.
type CreateListingPhotoInput struct {
	ObjectKey string
	AltText   string
	Position  int
}

func (service *ListingService) CreateListing(
	ctx context.Context,
	hostID uuid.UUID,
	input CreateListingInput,
) (Listing, error) {
	input, err := validateCreateInput(input)
	if err != nil {
		return Listing{}, err
	}
	now := service.now().UTC()
	listing := Listing{
		ID:               service.newID(),
		HostID:           hostID,
		Title:            input.Title,
		Description:      input.Description,
		Address:          input.Address,
		Type:             input.Type,
		PropertyType:     input.PropertyType,
		MaxGuests:        input.MaxGuests,
		MinimumAgeToBook: input.MinimumAgeToBook,
		Bedrooms:         input.Bedrooms,
		Beds:             input.Beds,
		Bathrooms:        input.Bathrooms,
		Amenities:        input.Amenities,
		Rules:            input.Rules,
		Status:           Draft,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	listing.Photos = make([]ListingPhoto, len(input.Photos))
	for index, photo := range input.Photos {
		listing.Photos[index] = ListingPhoto{
			ID: service.newID(), ObjectKey: photo.ObjectKey,
			AltText: photo.AltText, Position: photo.Position,
		}
	}

	// Save the listing as the authenticated host.
	err = service.store.Create(ctx, hostID.String(), listing)

	if err != nil {
		return Listing{}, err
	}

	return listing, nil
}

var ErrInvalidListing = errors.New("invalid listing")

func validateCreateInput(input CreateListingInput) (CreateListingInput, error) {
	input = trimCreateInput(input)

	if err := validateCoreFields(input); err != nil {
		return input, err
	}
	if err := validateAddress(input.Address); err != nil {
		return input, err
	}
	if err := validateCapacity(input); err != nil {
		return input, err
	}
	if err := validateCollectionSizes(input); err != nil {
		return input, err
	}

	amenities, err := dedupeAmenities(input.Amenities)
	if err != nil {
		return input, err
	}
	input.Amenities = amenities

	if err := validatePhotos(input.Photos); err != nil {
		return input, err
	}
	if err := validateRules(input.Rules); err != nil {
		return input, err
	}
	return input, nil
}

// trimCreateInput normalizes whitespace (and the country code's casing)
// before any length or content checks run.
func trimCreateInput(input CreateListingInput) CreateListingInput {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Address.Line1 = strings.TrimSpace(input.Address.Line1)
	input.Address.Line2 = strings.TrimSpace(input.Address.Line2)
	input.Address.City = strings.TrimSpace(input.Address.City)
	input.Address.Region = strings.TrimSpace(input.Address.Region)
	input.Address.PostalCode = strings.TrimSpace(input.Address.PostalCode)
	input.Address.CountryCode = strings.ToUpper(strings.TrimSpace(input.Address.CountryCode))
	return input
}

func validateCoreFields(input CreateListingInput) error {
	if len(input.Title) < 4 || len(input.Title) > 120 {
		return invalidListing("title must be 4-120 characters")
	}
	if len(input.Description) < 20 || len(input.Description) > 5000 {
		return invalidListing("description must be 20-5000 characters")
	}
	if !validListingType(input.Type) || !validPropertyType(input.PropertyType) {
		return invalidListing("unsupported listing or property type")
	}
	return nil
}

func validateAddress(address Address) error {
	if address.Line1 == "" || address.City == "" || len(address.CountryCode) != 2 {
		return invalidListing("line1, city, and a two-letter country_code are required")
	}
	if address.Latitude < -90 || address.Latitude > 90 || address.Longitude < -180 || address.Longitude > 180 {
		return invalidListing("latitude or longitude is out of range")
	}
	return nil
}

func validateCapacity(input CreateListingInput) error {
	if input.MaxGuests < 1 || input.MaxGuests > 50 {
		return invalidListing("max_guests must be between 1 and 50")
	}
	if input.MinimumAgeToBook < 18 || input.MinimumAgeToBook > 99 {
		return invalidListing("minimum_age_to_book must be between 18 and 99")
	}
	if input.Bedrooms < 0 || input.Bedrooms > 50 || input.Beds < 1 || input.Beds > 100 || input.Bathrooms < 0.5 || input.Bathrooms > 50 {
		return invalidListing("bedroom, bed, or bathroom count is out of range")
	}
	return nil
}

func validateCollectionSizes(input CreateListingInput) error {
	if len(input.Amenities) > 100 || len(input.Photos) > 100 || len(input.Rules) > 50 {
		return invalidListing("too many amenities, photos, or rules")
	}
	return nil
}

// dedupeAmenities rejects anything not in the amenity catalog and silently
// drops repeats, since a host resubmitting the same amenity twice isn't
// worth failing the request over.
func dedupeAmenities(input []Amenity) ([]Amenity, error) {
	seen := map[Amenity]struct{}{}
	amenities := make([]Amenity, 0, len(input))
	for _, amenity := range input {
		if !validAmenity(amenity) {
			return nil, invalidListing("unknown amenity %q", amenity)
		}
		if _, duplicate := seen[amenity]; duplicate {
			continue
		}
		seen[amenity] = struct{}{}
		amenities = append(amenities, amenity)
	}
	return amenities, nil
}

func validatePhotos(photos []CreateListingPhotoInput) error {
	seenPositions := map[int]struct{}{}
	for index := range photos {
		photo := &photos[index]
		photo.ObjectKey = strings.TrimSpace(photo.ObjectKey)
		photo.AltText = strings.TrimSpace(photo.AltText)
		if photo.ObjectKey == "" || len(photo.ObjectKey) > 1024 || len(photo.AltText) > 500 || photo.Position < 0 {
			return invalidListing("photo metadata is invalid")
		}
		if _, duplicate := seenPositions[photo.Position]; duplicate {
			return invalidListing("photo positions must be unique")
		}
		seenPositions[photo.Position] = struct{}{}
	}
	return nil
}

func validateRules(rules []HouseRule) error {
	for index := range rules {
		rules[index].Title = strings.TrimSpace(rules[index].Title)
		rules[index].Description = strings.TrimSpace(rules[index].Description)
		if rules[index].Title == "" || len(rules[index].Title) > 120 || len(rules[index].Description) > 1000 {
			return invalidListing("house rule is invalid")
		}
	}
	return nil
}

func invalidListing(format string, values ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidListing, fmt.Sprintf(format, values...))
}
func validListingType(value ListingType) bool {
	return value == EntirePlace || value == PrivateRoom || value == SharedRoom
}
func validPropertyType(value PropertyType) bool {
	return value == House || value == Apartment || value == Condo || value == Townhouse || value == Villa
}
