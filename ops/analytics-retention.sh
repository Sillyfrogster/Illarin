#!/bin/sh

# Deletes Umami rows older than 30 days, at start and nightly at 04:30 UTC

set -eu

run_at_seconds=$((4 * 3600 + 30 * 60))

delete_old_rows() {
  psql -v ON_ERROR_STOP=1 --quiet <<'SQL'
BEGIN;
SET LOCAL search_path TO umami;
CREATE TEMP TABLE cutoff ON COMMIT DROP AS SELECT now() - interval '30 days' AS at;
DELETE FROM event_data WHERE created_at < (SELECT at FROM cutoff);
DELETE FROM session_data WHERE created_at < (SELECT at FROM cutoff);
DELETE FROM revenue WHERE created_at < (SELECT at FROM cutoff);
DELETE FROM heatmap_event WHERE created_at < (SELECT at FROM cutoff);
DELETE FROM session_replay WHERE created_at < (SELECT at FROM cutoff);
DELETE FROM session_link WHERE created_at < (SELECT at FROM cutoff);
DELETE FROM website_event WHERE created_at < (SELECT at FROM cutoff);
DELETE FROM session
WHERE created_at < (SELECT at FROM cutoff)
  AND NOT EXISTS (SELECT FROM website_event WHERE website_event.session_id = session.session_id);
COMMIT;
SQL
  echo "Deleted analytics rows older than 30 days."
}

delete_old_rows
if [ "${1:-}" = "once" ]; then
  exit 0
fi

while true; do
  now=$(date -u +%s)
  wait_seconds=$(( (run_at_seconds - now % 86400 + 86400) % 86400 ))
  sleep "$(( wait_seconds == 0 ? 86400 : wait_seconds ))"
  delete_old_rows || echo "Deleting old analytics rows failed; trying again tomorrow." >&2
done
