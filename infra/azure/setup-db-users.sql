-- Run once as gydadmin after provisioning the PostgreSQL server.
-- Creates a restricted runtime user (gydapp) for the application.
--
-- Security model:
--   gydadmin  — used by CI migration step (goose up) and pgBouncer → Postgres
--   gydapp    — future runtime user once the app supports GYD_MIGRATION_DSN split;
--               has DML only — cannot CREATE/DROP tables or modify schema
--
-- To run via Azure CLI (no local psql needed):
--   az postgres flexible-server execute \
--     --name gyd-pg-main \
--     --resource-group gyd-persistent \
--     --admin-user gydadmin \
--     --admin-password '<password>' \
--     --database-name gatheryourdeals \
--     --file-path infra/azure/setup-db-users.sql

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'gydapp') THEN
    CREATE USER gydapp WITH PASSWORD 'REPLACE_WITH_GYDAPP_PASSWORD';
    RAISE NOTICE 'Created user gydapp';
  ELSE
    RAISE NOTICE 'User gydapp already exists — skipping CREATE';
  END IF;
END
$$;

GRANT CONNECT ON DATABASE gatheryourdeals TO gydapp;
GRANT USAGE ON SCHEMA public TO gydapp;

-- Grant on all currently existing tables (covers goose_db_version once migrations run).
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO gydapp;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO gydapp;

-- Grant on all future tables created by migrations.
ALTER DEFAULT PRIVILEGES FOR ROLE gydadmin IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO gydapp;
ALTER DEFAULT PRIVILEGES FOR ROLE gydadmin IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO gydapp;
