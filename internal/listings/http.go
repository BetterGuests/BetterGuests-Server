package listings

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/BetterGuests/BetterGuests-Server/internal/accounts"
	"github.com/google/uuid"
)

// ListingCreator is the one application capability the create-listing HTTP
// route needs. ListingService will implement it when CreateListing is built.
type ListingCreator interface {
	CreateListing(
		ctx context.Context,
		hostID uuid.UUID,
		input CreateListingInput,
	) (Listing, error)
}

// HTTPHandler owns the HTTP-facing listing routes. It knows only the
// application capability above, not PostgreSQL or object storage.
type HTTPHandler struct {
	listingCreator ListingCreator
}

func NewHTTPHandler(listingCreator ListingCreator) *HTTPHandler {
	return &HTTPHandler{listingCreator: listingCreator}
}

// CreateListingRequest is the JSON body accepted by POST /api/v1/listings.
// Host identity is deliberately absent because it comes from the validated
// bearer token.
type CreateListingRequest struct {
	Title            string                      `json:"title"`
	Description      string                      `json:"description"`
	Address          Address                     `json:"address"`
	Type             ListingType                 `json:"type"`
	PropertyType     PropertyType                `json:"property_type"`
	MaxGuests        int                         `json:"max_guests"`
	MinimumAgeToBook int                         `json:"minimum_age_to_book"`
	Bedrooms         int                         `json:"bedrooms"`
	Beds             int                         `json:"beds"`
	Bathrooms        float64                     `json:"bathrooms"`
	Amenities        []Amenity                   `json:"amenities"`
	Photos           []CreateListingPhotoRequest `json:"photos"`
	Rules            []HouseRule                 `json:"rules"`
}

type CreateListingPhotoRequest struct {
	ObjectKey string `json:"object_key"`
	AltText   string `json:"alt_text,omitempty"`
	Position  int    `json:"position"`
}

func (handler *HTTPHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
	hostID, ok := accounts.UserID(r.Context())

	if !ok {
		writeListingError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var request CreateListingRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeListingError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeListingError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return
	}

	photos := make([]CreateListingPhotoInput, len(request.Photos))
	for index, photo := range request.Photos {
		photos[index] = CreateListingPhotoInput(photo)
	}

	listing, err := handler.listingCreator.CreateListing(r.Context(), hostID, CreateListingInput{
		Title: request.Title, Description: request.Description, Address: request.Address,
		Type: request.Type, PropertyType: request.PropertyType, MaxGuests: request.MaxGuests,
		MinimumAgeToBook: request.MinimumAgeToBook, Bedrooms: request.Bedrooms,
		Beds: request.Beds, Bathrooms: request.Bathrooms, Amenities: request.Amenities,
		Photos: photos, Rules: request.Rules,
	})

	if errors.Is(err, ErrInvalidListing) {
		writeListingError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err != nil {
		writeListingError(w, http.StatusInternalServerError, "could not create listing")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/listings/"+listing.ID.String())
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(listing)
}

func writeListingError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
