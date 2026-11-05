#!/bin/sh
# One backup of PostgreSQL: every database (LibreQuenda's and Zitadel's, whose
# accounts the bookings point to) in pg_dump's custom format, and the roles,
# under <bucket>/postgres/<UTC time>/. Each dump is read back before it is
# uploaded; then backups older than BACKUP_KEEP_DAYS are removed.
# run-daily runs it every day; by hand: docker compose exec backup backup
set -eu

STATUS=/tmp/backup-status
stamp=$(date -u +%Y%m%dT%H%M%SZ)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

fail() {
	echo "[backup] $stamp FAILED: $*" >&2
	echo failed >"$STATUS"
	exit 1
}

export PGHOST=db PGUSER="$POSTGRES_USER" PGPASSWORD="$POSTGRES_PASSWORD"
target="backup/$BACKUP_BUCKET"

mcli alias set backup "$BACKUP_S3_URL" "$BACKUP_S3_ACCESS_KEY" "$BACKUP_S3_SECRET_KEY" --api s3v4 --path on >/dev/null ||
	fail "cannot set up the storage at $BACKUP_S3_URL"
mcli mb --ignore-existing --region "$BACKUP_S3_REGION" "$target" >/dev/null ||
	fail "cannot reach or create the bucket $BACKUP_BUCKET at $BACKUP_S3_URL"

databases=$(psql -d "$POSTGRES_DB" -Atc "SELECT datname FROM pg_database WHERE NOT datistemplate AND datname <> 'postgres' ORDER BY 1") ||
	fail "cannot list the databases"
[ -n "$databases" ] || fail "no databases to back up"

# roles and their password hashes: a fresh volume gets them from db/init, but
# not if the .env passwords have changed since
pg_dumpall --globals-only >"$work/globals.sql" || fail "cannot dump the roles"

for db in $databases; do
	pg_dump --format=custom --dbname="$db" --file="$work/$db.dump" || fail "cannot dump $db"
	pg_restore --list "$work/$db.dump" >/dev/null || fail "the dump of $db cannot be read back"
done

mcli cp --recursive --quiet "$work/" "$target/postgres/$stamp/" >/dev/null ||
	fail "cannot upload to $BACKUP_BUCKET/postgres/$stamp/"

# a failed clean-up keeps more backups than asked, never fewer: a warning only
mcli rm --recursive --force --older-than "${BACKUP_KEEP_DAYS}d" "$target/postgres/" >/dev/null 2>&1 ||
	echo "[backup] $stamp warning: cannot remove backups older than $BACKUP_KEEP_DAYS days" >&2

echo ok >"$STATUS"
echo "[backup] $stamp done: $(cd "$work" && du -ch -- * | tail -1 | cut -f1) in $BACKUP_BUCKET/postgres/$stamp/ ($(echo $databases | tr ' ' ','))"
