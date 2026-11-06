\ir ../bootstrap/001_roles_and_schema.sql

-- These credentials exist only inside the disposable local test container.
ALTER ROLE betterguests_migrator PASSWORD 'betterguests_migrator_test';
ALTER ROLE betterguests_app PASSWORD 'betterguests_app_test';
