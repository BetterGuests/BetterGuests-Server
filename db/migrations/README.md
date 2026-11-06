# Migration security contract

Application migrations run as `betterguests_migrator`. The API runs as
`betterguests_app`; it must never own database objects or receive schema
creation privileges.

Every tenant-owned table migration must, before the transaction commits:

1. Use the `app` schema and a UUID primary key.
2. Add explicit `NOT NULL`, foreign-key, unique, and check constraints.
3. Add indexes for foreign keys and expected filters.
4. Enable and force row-level security.
5. Create command-specific policies for `betterguests_app` using
   `app.current_user_id()`.
6. Use both `USING` and `WITH CHECK` where writes are allowed.
7. Explicitly grant only the operations the API needs.
8. Never grant `TRUNCATE`, `REFERENCES`, `TRIGGER`, ownership, or schema
   creation to the API role.

Example:

```sql
CREATE TABLE app.example (
    id uuid PRIMARY KEY,
    owner_id uuid NOT NULL,
    value text NOT NULL CHECK (length(value) <= 1000)
);

CREATE INDEX example_owner_id_idx ON app.example (owner_id);

ALTER TABLE app.example ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.example FORCE ROW LEVEL SECURITY;

CREATE POLICY example_select ON app.example
    FOR SELECT TO betterguests_app
    USING (owner_id = app.current_user_id());

CREATE POLICY example_insert ON app.example
    FOR INSERT TO betterguests_app
    WITH CHECK (owner_id = app.current_user_id());

CREATE POLICY example_update ON app.example
    FOR UPDATE TO betterguests_app
    USING (owner_id = app.current_user_id())
    WITH CHECK (owner_id = app.current_user_id());

CREATE POLICY example_delete ON app.example
    FOR DELETE TO betterguests_app
    USING (owner_id = app.current_user_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON app.example TO betterguests_app;
```

The integration tests query PostgreSQL's catalogs and fail when these
invariants are violated. Do not weaken those tests to make a migration pass.
