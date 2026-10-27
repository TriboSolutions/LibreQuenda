#!/bin/bash
# ============================================================
#  APP ROLE: the API connects as librequenda_app, not as the owner.
#  The owner (POSTGRES_USER) is exempt from row-level security;
#  this role is not, so the policies in 05_row_security.sql apply to it.
# ============================================================
set -e
: "${APP_DB_PASSWORD:?set APP_DB_PASSWORD in .env}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
     -v app_password="$APP_DB_PASSWORD" <<-'EOSQL'
    CREATE ROLE librequenda_app LOGIN PASSWORD :'app_password';

    -- data only: no DDL, no TRUNCATE, owns nothing
    GRANT USAGE ON SCHEMA public TO librequenda_app;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO librequenda_app;
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO librequenda_app;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO librequenda_app;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO librequenda_app;
EOSQL
