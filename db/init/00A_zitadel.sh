#!/bin/bash
# ============================================================
#  ZITADEL: own role + database, separate from the app database
# ============================================================
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE ROLE zitadel LOGIN PASSWORD '${ZITADEL_DB_PASSWORD}';
    CREATE DATABASE zitadel OWNER zitadel;
EOSQL
