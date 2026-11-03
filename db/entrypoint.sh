#!/bin/bash
set -e

STAGING_DIR="/docker-initdb-staging"
INIT_DIR="/docker-entrypoint-initdb.d"

cp -r "$STAGING_DIR/." "$INIT_DIR/"

if [ "${DB_SEED}" = "true" ] && [ -f "$INIT_DIR/09_initial_data.sql" ]; then
    echo "[entrypoint] DB_SEED=true — schema + seed data will be loaded"
elif [ "${DB_SEED}" = "true" ]; then
    echo "[entrypoint] DB_SEED=true but there is no seed data (generate it with db/init/generate_data/main.py, --profile dev or prod) — schema only"
else
    echo "[entrypoint] DB_SEED=false — schema only, removing seed data"
    rm -f "$INIT_DIR/09_initial_data.sql"
fi

exec docker-entrypoint.sh "$@"
