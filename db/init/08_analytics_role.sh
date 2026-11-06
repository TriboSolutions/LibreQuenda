#!/bin/bash
# ============================================================
#  ANALYTICS ROLE: librequenda_analytics reads the analytics views
#  (07_analytics_views.sql) and nothing else, read-only.
#  The API connects as it to run the SQL an assistant writes for the councils'
#  and the superadministrators' questions; the guarantees are here, not in
#  checking that SQL:
#    - SELECT on the analytics views only: no right on any table;
#    - every transaction read-only, with a time limit;
#    - no set_config(), so a query cannot widen the councils it sees
#      (analytics.tenants / analytics.superadmin, set by the API with SET LOCAL).
# ============================================================
set -e
: "${ANALYTICS_DB_PASSWORD:?set ANALYTICS_DB_PASSWORD in .env}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
     -v analytics_password="$ANALYTICS_DB_PASSWORD" <<-'EOSQL'
    CREATE ROLE librequenda_analytics LOGIN PASSWORD :'analytics_password' NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
    ALTER ROLE librequenda_analytics SET default_transaction_read_only = on;
    ALTER ROLE librequenda_analytics SET statement_timeout = '10s';
    ALTER ROLE librequenda_analytics SET search_path = analytics;

    GRANT USAGE ON SCHEMA analytics TO librequenda_analytics;
    GRANT SELECT ON ALL TABLES IN SCHEMA analytics TO librequenda_analytics;

    -- its councils are the API's to set: nothing it runs may change them
    REVOKE EXECUTE ON FUNCTION pg_catalog.set_config(TEXT, TEXT, BOOLEAN) FROM PUBLIC;
    GRANT EXECUTE ON FUNCTION pg_catalog.set_config(TEXT, TEXT, BOOLEAN) TO librequenda_app;
EOSQL
