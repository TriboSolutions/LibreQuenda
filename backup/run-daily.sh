#!/bin/sh
# Runs one backup a day at BACKUP_AT (HH:MM, in the container's TZ). The health
# check reads /tmp/backup-status: "failed" after a backup that failed, until
# the next one works; "pending" before the first.
set -u

case "$BACKUP_AT" in
[01][0-9]:[0-5][0-9] | 2[0-3]:[0-5][0-9]) ;;
*)
	echo "[backup] BACKUP_AT must be HH:MM, e.g. 03:30 (got \"$BACKUP_AT\")" >&2
	exit 1
	;;
esac
case "$BACKUP_KEEP_DAYS" in
'' | *[!0-9]* | 0)
	echo "[backup] BACKUP_KEEP_DAYS must be a number of days, at least 1 (got \"$BACKUP_KEEP_DAYS\")" >&2
	exit 1
	;;
esac

echo pending >/tmp/backup-status
echo "[backup] every day at $BACKUP_AT ($TZ) into $BACKUP_S3_URL/$BACKUP_BUCKET, kept $BACKUP_KEEP_DAYS days"

last=""
while :; do
	today=$(date +%F)
	if [ "$(date +%H:%M)" = "$BACKUP_AT" ] && [ "$last" != "$today" ]; then
		last=$today
		backup || true # logged, and the health check turns unhealthy
	fi
	sleep 20
done
