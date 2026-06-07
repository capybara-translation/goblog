#!/usr/bin/env bash
#
# backup-db.sh — Take a consistent snapshot of the goblog SQLite database and
# upload it (gzip-compressed) to S3. Designed to run once a day from a systemd
# timer (see deploy/goblog-backup.{service,timer}).
#
# Why not just `cp` the DB file? goblog runs in rollback-journal mode, so the
# file on disk may be mid-write at any instant; copying it can capture a torn,
# unrestorable image. SQLite's online backup (`.backup`) takes a transactionally
# consistent snapshot while the app keeps serving. We open the source read-only
# so the backup process can never modify the live database.
#
# Monitoring is AWS-native: on success the script emits a CloudWatch metric
# (BackupSuccess=1); on any failure it emits 0. A CloudWatch Alarm with
# TreatMissingData=breaching turns BOTH "ran and failed" (value 0) and "never
# ran / instance down" (no datapoint) into an alert via SNS. See docs/BACKUP.md.
#
# Configuration is read from the environment (see deploy/backup.env.example):
#   S3_BUCKET         (required)  target bucket, e.g. my-goblog-backups
#   S3_PREFIX         (optional)  key prefix, default "goblog"
#   DB_PATH           (optional)  source DB, default /var/lib/goblog/goblog.db
#   METRIC_NAMESPACE  (optional)  CloudWatch namespace, default "Goblog/Backup"
#   METRIC_NAME       (optional)  CloudWatch metric, default "BackupSuccess"
#   AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_DEFAULT_REGION (for aws cli)

set -Eeuo pipefail

# Config with safe defaults FIRST (these can never fail), so the EXIT trap
# installed below can still report via CloudWatch even if a required variable or
# mktemp later fails. METRIC_* must be set before the trap because emit_metric
# uses them.
DB_PATH="${DB_PATH:-/var/lib/goblog/goblog.db}"
S3_PREFIX="${S3_PREFIX:-goblog}"
METRIC_NAMESPACE="${METRIC_NAMESPACE:-Goblog/Backup}"
METRIC_NAME="${METRIC_NAME:-BackupSuccess}"

# emit_metric publishes the success/failure heartbeat to CloudWatch. It is
# best-effort (|| true): a metrics hiccup must never fail the backup itself, and
# a missing datapoint is fail-safe — the alarm's TreatMissingData=breaching
# turns silence into an alert anyway.
emit_metric() {
    aws cloudwatch put-metric-data \
        --namespace "$METRIC_NAMESPACE" \
        --metric-name "$METRIC_NAME" \
        --value "$1" --unit Count >/dev/null 2>&1 || true
}

# Single cleanup-and-report path. Fires on EVERY exit (success, `exit 1`, a
# set -e abort, or an unexpected error). On any non-zero exit it emits a 0
# heartbeat so a failed run ALERTS instead of being mistaken for "no run". This
# is deliberately an EXIT trap (not ERR) so an explicit `exit 1` in a validation
# guard still reports failure. Installed BEFORE the required-var check and mktemp
# so those failure paths are covered too; `work` may be unset, hence "${work:-}".
finish() {
    local rc=$?
    # Only clean up if mktemp already ran; never call `rm` with an empty path
    # (the trap can fire before $work is set, e.g. on the S3_BUCKET guard).
    [ -n "${work:-}" ] && rm -rf "$work"
    if [ "$rc" -ne 0 ]; then
        emit_metric 0
    fi
}
trap finish EXIT

# Things that can fail — now all covered by the trap above. Use an explicit
# exit (not ${S3_BUCKET:?...}): a parameter-expansion abort during assignment
# does not reliably propagate a non-zero status to the EXIT trap, so the 0
# heartbeat would be skipped. An explicit `exit 1` triggers finish() correctly.
if [ -z "${S3_BUCKET:-}" ]; then
    echo "S3_BUCKET is required" >&2
    exit 1
fi
work="$(mktemp -d)"

# Guard: the source must exist and be readable. Without this, sqlite3's `.open`
# silently falls back to an empty in-memory DB ("Notice: using substitute
# in-memory database") and we would upload a valid-looking but EMPTY backup and
# report success — silent data loss.
if [ ! -f "$DB_PATH" ] || [ ! -r "$DB_PATH" ]; then
    echo "source DB not found or not readable: $DB_PATH" >&2
    exit 1
fi

ts="$(date -u +%Y-%m-%dT%H%M%SZ)"
snap="$work/goblog-$ts.db"

# (1) Consistent, read-only online snapshot. `.open --readonly` guarantees we
#     never open the live DB read-write (also avoids EROFS under the service's
#     ProtectSystem=strict sandbox). `.timeout` waits out a transient writer
#     lock. `-bail` aborts on SQL execution errors (e.g. a `.backup` I/O error),
#     but it does NOT catch `.open` of a missing file: sqlite3 silently falls
#     back to an empty in-memory DB and exits 0. That case is caught instead by
#     the existence guard above and the 'posts' table check below — keep both.
sqlite3 -bail <<SQL
.open --readonly '$DB_PATH'
.timeout 60000
.backup '$snap'
SQL

# (2) Verify the snapshot before trusting it:
#     (a) structural integrity, and
#     (b) it actually contains the goblog schema — guards against having
#         snapshotted an empty/substitute/wrong DB even if step (1) "succeeded".
if ! sqlite3 "$snap" 'PRAGMA integrity_check;' | grep -qx 'ok'; then
    echo "integrity_check failed for $snap" >&2
    exit 1
fi
if [ "$(sqlite3 "$snap" "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='posts';")" != "1" ]; then
    echo "snapshot is missing the expected 'posts' table — refusing to upload $snap" >&2
    exit 1
fi

# (3) Compress, then (4) upload. Keys are partitioned by year/month for sane
#     listing and lifecycle rules. S3 PutObject is atomic, so a partial upload
#     never appears as a usable object.
gzip -9 "$snap"
# Partition the key by year/month, derived from $ts (a single clock read) so the
# filename and prefix can never disagree across a month boundary.
ymd="${ts%%T*}"   # YYYY-MM-DD
key="$S3_PREFIX/${ymd:0:4}/${ymd:5:2}/goblog-$ts.db.gz"
aws s3 cp "$snap.gz" "s3://$S3_BUCKET/$key" --only-show-errors

echo "backup uploaded: s3://$S3_BUCKET/$key"

# (5) Success heartbeat (only reached when everything above succeeded).
emit_metric 1
