-- Administrator bootstrap: run once per database before normal migrations.
-- Passwords are deliberately set out-of-band and never stored here.

DO $roles$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'betterguests_migrator') THEN
        CREATE ROLE betterguests_migrator
            LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
            NOREPLICATION NOBYPASSRLS;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'betterguests_app') THEN
        CREATE ROLE betterguests_app
            LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
            NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 20;
    END IF;
END
$roles$;

ALTER ROLE betterguests_migrator
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
ALTER ROLE betterguests_app
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS
    CONNECTION LIMIT 20;

DO $database_privileges$
BEGIN
    EXECUTE format('REVOKE CONNECT, TEMPORARY ON DATABASE %I FROM PUBLIC', current_database());
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO betterguests_migrator, betterguests_app',
        current_database()
    );
END
$database_privileges$;

REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM betterguests_migrator, betterguests_app;

CREATE SCHEMA IF NOT EXISTS app AUTHORIZATION betterguests_migrator;
ALTER SCHEMA app OWNER TO betterguests_migrator;
REVOKE ALL ON SCHEMA app FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA app TO betterguests_migrator;
GRANT USAGE ON SCHEMA app TO betterguests_app;
REVOKE CREATE ON SCHEMA app FROM betterguests_app;

-- Undo the older broad defaults. Every application table must be granted
-- explicitly only after its RLS policies exist.
ALTER DEFAULT PRIVILEGES FOR ROLE betterguests_migrator IN SCHEMA public
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM betterguests_app;
ALTER DEFAULT PRIVILEGES FOR ROLE betterguests_migrator IN SCHEMA public
    REVOKE USAGE, SELECT ON SEQUENCES FROM betterguests_app;
ALTER DEFAULT PRIVILEGES FOR ROLE betterguests_migrator IN SCHEMA app
    REVOKE ALL ON TABLES FROM PUBLIC, betterguests_app;
ALTER DEFAULT PRIVILEGES FOR ROLE betterguests_migrator IN SCHEMA app
    REVOKE ALL ON SEQUENCES FROM PUBLIC, betterguests_app;
ALTER DEFAULT PRIVILEGES FOR ROLE betterguests_migrator
    REVOKE EXECUTE ON ROUTINES FROM PUBLIC;

ALTER ROLE betterguests_app SET search_path = app, pg_catalog;
ALTER ROLE betterguests_app SET statement_timeout = '5s';
ALTER ROLE betterguests_app SET lock_timeout = '2s';
ALTER ROLE betterguests_app SET idle_in_transaction_session_timeout = '10s';

ALTER ROLE betterguests_migrator SET search_path = app, pg_catalog;
ALTER ROLE betterguests_migrator SET statement_timeout = '5min';
ALTER ROLE betterguests_migrator SET lock_timeout = '5s';
ALTER ROLE betterguests_migrator SET idle_in_transaction_session_timeout = '1min';
