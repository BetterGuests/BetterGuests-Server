package listings

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// T Y P E S --------------------------------------- //

type Address struct {
	Line1       string  `json:"line1"`
	Line2       string  `json:"line2,omitempty"`
	City        string  `json:"city"`
	Region      string  `json:"region,omitempty"`
	PostalCode  string  `json:"postal_code,omitempty"`
	CountryCode string  `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	PlaceID     string  `json:"place_id,omitempty"`
}

type Listing struct {
	ID               uuid.UUID      `json:"id"`
	HostID           uuid.UUID      `json:"host_id"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Address          Address        `json:"address"`
	Type             ListingType    `json:"type"`
	PropertyType     PropertyType   `json:"property_type"`
	MaxGuests        int            `json:"max_guests"`
	MinimumAgeToBook int            `json:"minimum_age_to_book"`
	Bedrooms         int            `json:"bedrooms"`
	Beds             int            `json:"beds"`
	Bathrooms        float64        `json:"bathrooms"`
	Amenities        []Amenity      `json:"amenities"`
	Photos           []ListingPhoto `json:"photos"`
	Rules            []HouseRule    `json:"rules"`
	Status           ListingStatus  `json:"status"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	ReviewCount      int            `json:"review_count"`
	Reviews          []Review       `json:"reviews,omitempty"`
}

type ListingPhoto struct {
	ID uuid.UUID `json:"id"`

	// ObjectKey is the permanent identifier stored in the database. It is
	// never returned to clients; URL is generated from it when serving a
	// listing and expires according to the object-storage configuration.
	ObjectKey string `json:"-"`
	URL       string `json:"url,omitempty"`
	AltText   string `json:"alt_text,omitempty"`
	Position  int    `json:"position"`
}

type HouseRule struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type Amenity string

const (
	// Common amenities.
	BBQArea                Amenity = "bbq_area"
	FreeParking            Amenity = "free_parking"
	OutdoorKitchen         Amenity = "outdoor_kitchen"
	Washer                 Amenity = "washer"
	Dryer                  Amenity = "dryer"
	CarbonMonoxideDetector Amenity = "carbon_monoxide_detector"
	FireExtinguisher       Amenity = "fire_extinguisher"
	FirstAidKit            Amenity = "first_aid_kit"
	CleaningSupplies       Amenity = "cleaning_supplies"
	Dishwasher             Amenity = "dishwasher"
	StreetParking          Amenity = "street_parking"
	HairDryer              Amenity = "hair_dryer"
	HotWater               Amenity = "hot_water"
	AirConditioning        Amenity = "air_conditioning"

	// Bathrooms.
	Bathtub     Amenity = "bathtub"
	ShowerGel   Amenity = "shower_gel"
	BodySoap    Amenity = "body_soap"
	BathLinens  Amenity = "bath_linens"
	Conditioner Amenity = "conditioner"
	Shampoo     Amenity = "shampoo"

	// Patio.
	BBQUtensils    Amenity = "bbq_utensils"
	FirePit        Amenity = "fire_pit"
	OutdoorSeating Amenity = "outdoor_seating"
	PatioAmenity   Amenity = "patio"

	// Kitchen.
	BakingSheet    Amenity = "baking_sheet"
	Blender        Amenity = "blender"
	Coffee         Amenity = "coffee"
	CoffeeMaker    Amenity = "coffee_maker"
	CookingBasics  Amenity = "cooking_basics"
	Dishes         Amenity = "dishes"
	Freezer        Amenity = "freezer"
	IceMaker       Amenity = "ice_maker"
	Kettle         Amenity = "kettle"
	KitchenAmenity Amenity = "kitchen"
	Microwave      Amenity = "microwave"
	MiniFridge     Amenity = "mini_fridge"
	Oven           Amenity = "oven"
	Fridge         Amenity = "fridge"
	Stove          Amenity = "stove"
	Toaster        Amenity = "toaster"
	WineGlasses    Amenity = "wine_glasses"

	// Living room.
	BoardGames Amenity = "board_games"
	CeilingFan Amenity = "ceiling_fan"
	Heating    Amenity = "heating"
	Fireplace  Amenity = "fireplace"
	TV         Amenity = "tv"

	// Office and dining room.
	DedicatedWorkspace Amenity = "dedicated_workspace"
	DiningTable        Amenity = "dining_table"
	HighChair          Amenity = "high_chair"

	// Property areas.
	GarageAmenity Amenity = "garage"
	GardenAmenity Amenity = "garden"
	HotTub        Amenity = "hot_tub"
	OutdoorPool   Amenity = "outdoor_pool"

	// Common area and laundry room.
	WiFi                 Amenity = "wifi"
	LongTermStaysAllowed Amenity = "long_term_stays_allowed"
	SmokeDetector        Amenity = "smoke_detector"
	Iron                 Amenity = "iron"
	LaundromatNearby     Amenity = "laundromat_nearby"

	// Entry and exterior.
	LuggageDropOffAllowed Amenity = "luggage_drop_off_allowed"
	WheelchairAccessible  Amenity = "wheelchair_accessible"

	// Bedrooms.
	BedLinens    Amenity = "bed_linens"
	Safe         Amenity = "safe"
	Closet       Amenity = "closet"
	ExtraPillows Amenity = "extra_pillows"
	Hangers      Amenity = "hangers"
)

var CurrentAmenities = []Amenity{
	BBQArea, FreeParking, OutdoorKitchen, Washer, Dryer,
	CarbonMonoxideDetector, FireExtinguisher, FirstAidKit, CleaningSupplies,
	Dishwasher, StreetParking, HairDryer, HotWater, AirConditioning,
	Bathtub, ShowerGel, BodySoap, BathLinens, Conditioner, Shampoo,
	BBQUtensils, FirePit, OutdoorSeating, PatioAmenity,
	BakingSheet, Blender, Coffee, CoffeeMaker, CookingBasics, Dishes,
	Freezer, IceMaker, Kettle, KitchenAmenity, Microwave, MiniFridge,
	Oven, Fridge, Stove, Toaster, WineGlasses,
	BoardGames, CeilingFan, Heating, Fireplace, TV,
	DedicatedWorkspace, DiningTable, HighChair,
	GarageAmenity, GardenAmenity, HotTub, OutdoorPool,
	WiFi, LongTermStaysAllowed, SmokeDetector, Iron, LaundromatNearby,
	LuggageDropOffAllowed, WheelchairAccessible,
	BedLinens, Safe, Closet, ExtraPillows, Hangers,
}

func validAmenity(value Amenity) bool {
	return slices.Contains(CurrentAmenities, value)
}

type PropertyType string

const (
	House     PropertyType = "house"
	Apartment PropertyType = "apartment"
	Condo     PropertyType = "condo"
	Townhouse PropertyType = "townhouse"
	Villa     PropertyType = "villa"
)

type ListingStatus string

const (
	Draft     ListingStatus = "draft"
	Published ListingStatus = "published"
	Paused    ListingStatus = "paused"
)

type ListingType string

const (
	EntirePlace ListingType = "entire_place"
	PrivateRoom ListingType = "private_room"
	SharedRoom  ListingType = "shared_room"
)

type Review struct {
	ID        uuid.UUID `json:"id"`
	ListingID uuid.UUID `json:"listing_id"`
	BookingID uuid.UUID `json:"booking_id"`
	GuestID   uuid.UUID `json:"guest_id"`

	Rating       int    `json:"rating"`
	Comment      string `json:"comment"`
	LengthOfStay string `json:"length_of_stay"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
