-- +goose Up

-- ============================================================
-- Accounts
-- ============================================================

CREATE TABLE app.accounts (
    id          uuid PRIMARY KEY,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- ============================================================
-- Amenities (static reference table)
-- ============================================================

CREATE TABLE app.amenities (
    key         text PRIMARY KEY CHECK (key ~ '^[a-z0-9_]+$' AND length(key) <= 64),
    title       text NOT NULL CHECK (length(title) BETWEEN 1 AND 120),
    description text CHECK (description IS NULL OR length(description) <= 1000)
);

INSERT INTO app.amenities (key, title) VALUES
    -- Outdoor / parking
    ('bbq_area', 'BBQ area'),
    ('free_parking', 'Free parking'),
    ('outdoor_kitchen', 'Outdoor kitchen'),
    ('street_parking', 'Street parking'),
    ('bbq_utensils', 'BBQ utensils'),
    ('fire_pit', 'Fire pit'),
    ('outdoor_seating', 'Outdoor seating'),
    ('patio', 'Patio'),
    ('garage', 'Garage'),
    ('garden', 'Garden'),
    ('hot_tub', 'Hot tub'),
    ('outdoor_pool', 'Outdoor pool'),

    -- Laundry / safety
    ('washer', 'Washer'),
    ('dryer', 'Dryer'),
    ('carbon_monoxide_detector', 'Carbon monoxide detector'),
    ('fire_extinguisher', 'Fire extinguisher'),
    ('first_aid_kit', 'First aid kit'),
    ('cleaning_supplies', 'Cleaning supplies'),
    ('smoke_detector', 'Smoke detector'),
    ('iron', 'Iron'),
    ('laundromat_nearby', 'Laundromat nearby'),

    -- Bathroom
    ('hair_dryer', 'Hair dryer'),
    ('hot_water', 'Hot water'),
    ('air_conditioning', 'Air conditioning'),
    ('bathtub', 'Bathtub'),
    ('shower_gel', 'Shower gel'),
    ('body_soap', 'Body soap'),
    ('bath_linens', 'Bath linens'),
    ('conditioner', 'Conditioner'),
    ('shampoo', 'Shampoo'),

    -- Kitchen / dining
    ('dishwasher', 'Dishwasher'),
    ('baking_sheet', 'Baking sheet'),
    ('blender', 'Blender'),
    ('coffee', 'Coffee'),
    ('coffee_maker', 'Coffee maker'),
    ('cooking_basics', 'Cooking basics'),
    ('dishes', 'Dishes'),
    ('freezer', 'Freezer'),
    ('ice_maker', 'Ice maker'),
    ('kettle', 'Kettle'),
    ('kitchen', 'Kitchen'),
    ('microwave', 'Microwave'),
    ('mini_fridge', 'Mini fridge'),
    ('oven', 'Oven'),
    ('fridge', 'Fridge'),
    ('stove', 'Stove'),
    ('toaster', 'Toaster'),
    ('wine_glasses', 'Wine glasses'),
    ('dining_table', 'Dining table'),
    ('high_chair', 'High chair'),

    -- Living / common areas
    ('board_games', 'Board games'),
    ('ceiling_fan', 'Ceiling fan'),
    ('heating', 'Heating'),
    ('fireplace', 'Fireplace'),
    ('tv', 'TV'),
    ('dedicated_workspace', 'Dedicated workspace'),
    ('wifi', 'Wi-Fi'),
    ('long_term_stays_allowed', 'Long-term stays allowed'),

    -- Access / accessibility
    ('luggage_drop_off_allowed', 'Luggage drop-off allowed'),
    ('wheelchair_accessible', 'Wheelchair accessible'),

    -- Bedroom
    ('bed_linens', 'Bed linens'),
    ('safe', 'Safe'),
    ('closet', 'Closet'),
    ('extra_pillows', 'Extra pillows'),
    ('hangers', 'Hangers');

-- ============================================================
-- Listings
-- ============================================================

CREATE TABLE app.listings (
    id                   uuid PRIMARY KEY,
    host_id              uuid NOT NULL REFERENCES app.accounts(id),

    title                text NOT NULL CHECK (length(title) BETWEEN 4 AND 120),
    description          text NOT NULL CHECK (length(description) BETWEEN 20 AND 5000),

    address_line1        text NOT NULL CHECK (length(address_line1) <= 300),
    address_line2        text CHECK (address_line2 IS NULL OR length(address_line2) <= 300),
    city                 text NOT NULL CHECK (length(city) <= 120),
    region               text CHECK (region IS NULL OR length(region) <= 120),
    postal_code          text CHECK (postal_code IS NULL OR length(postal_code) <= 32),
    country_code         text NOT NULL CHECK (country_code ~ '^[A-Z]{2}$'),
    latitude             double precision NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude            double precision NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    place_id             text CHECK (place_id IS NULL OR length(place_id) <= 500),

    listing_type         text NOT NULL CHECK (listing_type IN ('entire_place', 'private_room', 'shared_room')),
    property_type        text NOT NULL CHECK (property_type IN ('house', 'apartment', 'condo', 'townhouse', 'villa')),

    max_guests           integer NOT NULL CHECK (max_guests BETWEEN 1 AND 50),
    minimum_age_to_book  integer NOT NULL CHECK (minimum_age_to_book BETWEEN 18 AND 99),
    bedrooms             integer NOT NULL CHECK (bedrooms BETWEEN 0 AND 50),
    beds                 integer NOT NULL CHECK (beds BETWEEN 1 AND 100),
    bathrooms            numeric(4,1) NOT NULL CHECK (bathrooms BETWEEN 0.5 AND 50),

    status               text NOT NULL CHECK (status IN ('draft', 'published', 'paused')),
    created_at           timestamptz NOT NULL,
    updated_at           timestamptz NOT NULL
);

CREATE INDEX listings_host_id_idx ON app.listings (host_id);
CREATE INDEX listings_status_idx ON app.listings (status);

-- ============================================================
-- Listing photos
-- ============================================================

CREATE TABLE app.listing_photos (
    id          uuid PRIMARY KEY,
    listing_id  uuid NOT NULL REFERENCES app.listings(id) ON DELETE CASCADE,
    object_key  text NOT NULL CHECK (length(object_key) BETWEEN 1 AND 1024),
    alt_text    text CHECK (alt_text IS NULL OR length(alt_text) <= 500),
    position    integer NOT NULL CHECK (position >= 0),

    UNIQUE (listing_id, position)
);

CREATE INDEX listing_photos_listing_id_idx ON app.listing_photos (listing_id);

-- ============================================================
-- Listing amenities (join table)
-- ============================================================

CREATE TABLE app.listing_amenities (
    id           uuid PRIMARY KEY,
    listing_id   uuid NOT NULL REFERENCES app.listings(id) ON DELETE CASCADE,
    amenity_key  text NOT NULL REFERENCES app.amenities(key),

    UNIQUE (listing_id, amenity_key)
);

CREATE INDEX listing_amenities_listing_id_idx ON app.listing_amenities (listing_id);
CREATE INDEX listing_amenities_key_idx ON app.listing_amenities (amenity_key);

-- ============================================================
-- Listing rules
-- ============================================================

CREATE TABLE app.listing_rules (
    id          uuid PRIMARY KEY,
    listing_id  uuid NOT NULL REFERENCES app.listings(id) ON DELETE CASCADE,
    title       text NOT NULL CHECK (length(title) BETWEEN 1 AND 120),
    description text CHECK (description IS NULL OR length(description) <= 1000),
    position    integer NOT NULL CHECK (position >= 0),

    UNIQUE (listing_id, position)
);

CREATE INDEX listing_rules_listing_id_idx ON app.listing_rules (listing_id);

-- ============================================================
-- Bookings
-- ============================================================

CREATE TABLE app.bookings (
    id            uuid PRIMARY KEY,
    listing_id    uuid NOT NULL REFERENCES app.listings(id),
    guest_id      uuid NOT NULL REFERENCES app.accounts(id),

    status        text NOT NULL CHECK (status IN ('pending', 'confirmed', 'cancelled', 'completed')),
    starts_on     date NOT NULL,
    ends_on       date NOT NULL CHECK (ends_on > starts_on),
    completed_at  timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),

    CHECK ((status = 'completed') = (completed_at IS NOT NULL))
);

CREATE INDEX bookings_listing_id_idx ON app.bookings (listing_id);
CREATE INDEX bookings_guest_id_idx ON app.bookings (guest_id);

-- ============================================================
-- Listing reviews
-- ============================================================

CREATE TABLE app.listing_reviews (
    id          uuid PRIMARY KEY,
    listing_id  uuid NOT NULL REFERENCES app.listings(id) ON DELETE CASCADE,
    guest_id    uuid NOT NULL REFERENCES app.accounts(id),
    booking_id  uuid NOT NULL UNIQUE REFERENCES app.bookings(id),

    rating      integer NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment     text NOT NULL CHECK (length(comment) BETWEEN 1 AND 4000),
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL
);

CREATE INDEX listing_reviews_listing_id_idx ON app.listing_reviews (listing_id);
CREATE INDEX listing_reviews_guest_id_idx ON app.listing_reviews (guest_id);

-- ============================================================
-- Row level security
-- ============================================================

ALTER TABLE app.accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.accounts FORCE ROW LEVEL SECURITY;

ALTER TABLE app.amenities ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.amenities FORCE ROW LEVEL SECURITY;

ALTER TABLE app.listings ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.listings FORCE ROW LEVEL SECURITY;

ALTER TABLE app.listing_photos ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.listing_photos FORCE ROW LEVEL SECURITY;

ALTER TABLE app.listing_amenities ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.listing_amenities FORCE ROW LEVEL SECURITY;

ALTER TABLE app.listing_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.listing_rules FORCE ROW LEVEL SECURITY;

ALTER TABLE app.bookings ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.bookings FORCE ROW LEVEL SECURITY;

ALTER TABLE app.listing_reviews ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.listing_reviews FORCE ROW LEVEL SECURITY;

-- ---- accounts -------------------------------------------------

CREATE POLICY accounts_select ON app.accounts
    FOR SELECT TO betterguests_app
    USING (id = app.current_user_id());

CREATE POLICY accounts_insert ON app.accounts
    FOR INSERT TO betterguests_app
    WITH CHECK (id = app.current_user_id());

-- ---- amenities --------------------------------------------------

CREATE POLICY amenities_select ON app.amenities
    FOR SELECT TO betterguests_app
    USING (true);

-- ---- listings -----------------------------------------------------

CREATE POLICY listings_select ON app.listings
    FOR SELECT TO betterguests_app
    USING (
        status = 'published'
        OR host_id = app.current_user_id()
    );

CREATE POLICY listings_insert ON app.listings
    FOR INSERT TO betterguests_app
    WITH CHECK (host_id = app.current_user_id());

CREATE POLICY listings_update ON app.listings
    FOR UPDATE TO betterguests_app
    USING (host_id = app.current_user_id())
    WITH CHECK (host_id = app.current_user_id());

CREATE POLICY listings_delete ON app.listings
    FOR DELETE TO betterguests_app
    USING (host_id = app.current_user_id());

-- ---- listing_photos -------------------------------------------------

CREATE POLICY listing_photos_select ON app.listing_photos
    FOR SELECT TO betterguests_app
    USING (
        EXISTS (SELECT 1 FROM app.listings l WHERE l.id = listing_id)
    );

CREATE POLICY listing_photos_insert ON app.listing_photos
    FOR INSERT TO betterguests_app
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

CREATE POLICY listing_photos_update ON app.listing_photos
    FOR UPDATE TO betterguests_app
    USING (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

CREATE POLICY listing_photos_delete ON app.listing_photos
    FOR DELETE TO betterguests_app
    USING (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

-- ---- listing_amenities ----------------------------------------------

CREATE POLICY listing_amenities_select ON app.listing_amenities
    FOR SELECT TO betterguests_app
    USING (
        EXISTS (SELECT 1 FROM app.listings l WHERE l.id = listing_id)
    );

CREATE POLICY listing_amenities_insert ON app.listing_amenities
    FOR INSERT TO betterguests_app
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

CREATE POLICY listing_amenities_delete ON app.listing_amenities
    FOR DELETE TO betterguests_app
    USING (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

-- ---- listing_rules ---------------------------------------------------

CREATE POLICY listing_rules_select ON app.listing_rules
    FOR SELECT TO betterguests_app
    USING (
        EXISTS (SELECT 1 FROM app.listings l WHERE l.id = listing_id)
    );

CREATE POLICY listing_rules_insert ON app.listing_rules
    FOR INSERT TO betterguests_app
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

CREATE POLICY listing_rules_update ON app.listing_rules
    FOR UPDATE TO betterguests_app
    USING (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

CREATE POLICY listing_rules_delete ON app.listing_rules
    FOR DELETE TO betterguests_app
    USING (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

-- ---- bookings ------------------------------------------------------

CREATE POLICY bookings_select ON app.bookings
    FOR SELECT TO betterguests_app
    USING (
        guest_id = app.current_user_id()
        OR EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

CREATE POLICY bookings_insert ON app.bookings
    FOR INSERT TO betterguests_app
    WITH CHECK (
        guest_id = app.current_user_id()
        AND status = 'pending'
        AND completed_at IS NULL
        AND EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.status = 'published'
        )
    );

CREATE POLICY bookings_update_host ON app.bookings
    FOR UPDATE TO betterguests_app
    USING (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM app.listings l
            WHERE l.id = listing_id AND l.host_id = app.current_user_id()
        )
    );

-- ---- listing_reviews ------------------------------------------------

CREATE POLICY reviews_select ON app.listing_reviews
    FOR SELECT TO betterguests_app
    USING (
        guest_id = app.current_user_id()
        OR EXISTS (SELECT 1 FROM app.listings l WHERE l.id = listing_id)
    );

CREATE POLICY reviews_insert ON app.listing_reviews
    FOR INSERT TO betterguests_app
    WITH CHECK (
        guest_id = app.current_user_id()
        AND EXISTS (
            SELECT 1 FROM app.bookings b
            WHERE b.id = booking_id
              AND b.guest_id = app.current_user_id()
              AND b.listing_id = listing_id
              AND b.status = 'completed'
        )
    );

CREATE POLICY reviews_update ON app.listing_reviews
    FOR UPDATE TO betterguests_app
    USING (guest_id = app.current_user_id())
    WITH CHECK (guest_id = app.current_user_id());

CREATE POLICY reviews_delete ON app.listing_reviews
    FOR DELETE TO betterguests_app
    USING (guest_id = app.current_user_id());

-- ============================================================
-- Grants
-- ============================================================

GRANT SELECT, INSERT ON app.accounts TO betterguests_app;
GRANT SELECT ON app.amenities TO betterguests_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON app.listings, app.listing_photos, app.listing_rules TO betterguests_app;
GRANT SELECT, INSERT, DELETE ON app.listing_amenities TO betterguests_app;
GRANT SELECT, INSERT, UPDATE ON app.bookings TO betterguests_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON app.listing_reviews TO betterguests_app;

-- +goose Down

DROP TABLE IF EXISTS app.listing_reviews;
DROP TABLE IF EXISTS app.bookings;
DROP TABLE IF EXISTS app.listing_rules;
DROP TABLE IF EXISTS app.listing_amenities;
DROP TABLE IF EXISTS app.listing_photos;
DROP TABLE IF EXISTS app.listings;
DROP TABLE IF EXISTS app.amenities;
DROP TABLE IF EXISTS app.accounts;
