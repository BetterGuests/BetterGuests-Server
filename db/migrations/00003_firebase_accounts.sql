-- +goose Up

-- Firebase issues string UIDs, while app.accounts uses UUID primary keys. The
-- API derives the UUID from the UID (see accounts.DeterministicID) and stores
-- the original UID here for traceability. The column stays nullable so rows
-- created before this migration remain valid.
ALTER TABLE app.accounts
    ADD COLUMN firebase_uid text
    CHECK (firebase_uid IS NULL OR length(firebase_uid) BETWEEN 1 AND 128);

CREATE UNIQUE INDEX accounts_firebase_uid_idx
    ON app.accounts (firebase_uid)
    WHERE firebase_uid IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS app.accounts_firebase_uid_idx;

ALTER TABLE app.accounts DROP COLUMN IF EXISTS firebase_uid;
