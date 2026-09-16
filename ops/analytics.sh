#!/usr/bin/env bash

set -euo pipefail

OPS_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$OPS_DIR/lib.sh"

# Creates the Umami database role and schema
prepare() {
  compose exec -T -e UMAMI_DATABASE_PASSWORD db \
    sh -c 'psql -v ON_ERROR_STOP=1 --quiet -U "$POSTGRES_USER" -d "$POSTGRES_DB"' <<'SQL'
\getenv password UMAMI_DATABASE_PASSWORD
SELECT format('CREATE ROLE umami LOGIN PASSWORD %L', :'password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'umami') \gexec
ALTER ROLE umami PASSWORD :'password';
CREATE SCHEMA IF NOT EXISTS umami AUTHORIZATION umami;
CREATE EXTENSION IF NOT EXISTS pgcrypto SCHEMA umami;
SQL
}

# Sets the Umami admin password and creates the website the site reports to
setup() {
  local site_host website_id
  site_host="$(host_of "${SITE_URL:?Set SITE_URL}")"
  website_id="$(compose exec -T umami curl -fsS -H "Host: $site_host" http://gateway:8080/ \
    | grep -m 1 -o 'data-website-id="[0-9a-f-]*"' | cut -d '"' -f 2 || true)"
  if [[ -z "$website_id" ]]; then
    echo "The site's front page carries no analytics website id." >&2
    exit 1
  fi
  : "${UMAMI_ADMIN_PASSWORD:?Set UMAMI_ADMIN_PASSWORD}"
  compose exec -T -e UMAMI_ADMIN_PASSWORD \
    -e "ANALYTICS_WEBSITE_ID=$website_id" -e "ANALYTICS_DOMAIN=${site_host%%:*}" \
    umami node --input-type=module - <"$OPS_DIR/umami-setup.mjs"
}

case "${1:-}" in
  prepare) prepare ;;
  setup) setup ;;
  *)
    echo "Usage: $0 prepare|setup" >&2
    exit 2
    ;;
esac
