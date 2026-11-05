#!/bin/sh
# Restores a backup made by backup.sh: each database in it is dropped and
# created again from its dump. Nothing may be connected to them meanwhile:
#   docker compose stop api zitadel-api zitadel-login
#   docker compose exec backup restore                    # lists the backups
#   docker compose exec backup restore 20261105T023000Z   # restores that one
#   docker compose start api zitadel-api zitadel-login
# The roles come from db/init with the .env passwords; the backup's
# globals.sql has them too, for a server set up with other passwords.
set -eu

fail() {
	echo "[restore] FAILED: $*" >&2
	exit 1
}

export PGHOST=db PGUSER="$POSTGRES_USER" PGPASSWORD="$POSTGRES_PASSWORD"
target="backup/$BACKUP_BUCKET"
mcli alias set backup "$BACKUP_S3_URL" "$BACKUP_S3_ACCESS_KEY" "$BACKUP_S3_SECRET_KEY" --api s3v4 --path on >/dev/null ||
	fail "cannot set up the storage at $BACKUP_S3_URL"

if [ $# -eq 0 ]; then
	echo "Backups in $BACKUP_BUCKET (restore one with: restore <name>):"
	mcli ls "$target/postgres/" | awk '{print $NF}' | tr -d /
	exit 0
fi

stamp=$1
case "$stamp" in
'' | *[!0-9TZ]*) fail "\"$stamp\" is not a backup's name, e.g. 20261105T023000Z (run restore alone to list them)" ;;
esac

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mcli cp --recursive --quiet "$target/postgres/$stamp/" "$work/" >/dev/null 2>&1 || fail "cannot download $BACKUP_BUCKET/postgres/$stamp/"
dumps=$(ls "$work"/*.dump 2>/dev/null) || fail "no backup named $stamp"

names=""
for dump in $dumps; do
	pg_restore --list "$dump" >/dev/null || fail "$(basename "$dump") cannot be read"
	names="$names${names:+,}'$(basename "$dump" .dump)'"
done

busy=$(psql -d postgres -Atc "SELECT string_agg(DISTINCT datname || ' (' || COALESCE(NULLIF(application_name, ''), usename) || ')', ', ') FROM pg_stat_activity WHERE datname IN ($names) AND pid <> pg_backend_pid()") ||
	fail "cannot reach PostgreSQL"
[ -z "$busy" ] || fail "still connected: $busy. Stop api, zitadel-api and zitadel-login first"

for dump in $dumps; do
	echo "[restore] $(basename "$dump" .dump) from $stamp"
	pg_restore --clean --if-exists --create --exit-on-error --dbname=postgres "$dump" || fail "restoring $(basename "$dump")"
done
echo "[restore] done: start api, zitadel-api and zitadel-login again"
