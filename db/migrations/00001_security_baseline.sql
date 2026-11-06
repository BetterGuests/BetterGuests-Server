-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION app.current_user_id()
RETURNS uuid
LANGUAGE sql
STABLE
PARALLEL SAFE
SET search_path = pg_catalog
AS $$
    SELECT NULLIF(current_setting('app.user_id', true), '')::uuid
$$;

REVOKE ALL ON FUNCTION app.current_user_id() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.current_user_id() TO betterguests_app;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.current_user_id();
