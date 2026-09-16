#!/usr/bin/env bash

set -euo pipefail

OPS_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$OPS_DIR/.." && pwd)"
TEST_DIR="$(mktemp -d /tmp/illarin-stack.XXXXXX)"
TEST_ENV="$TEST_DIR/production.env"
MICROSOFT_TEST_ENV="$TEST_DIR/microsoft-production.env"
TEST_PROJECT="illarin-smoke"
TEST_PORT="${ILLARIN_TEST_PORT:-18080}"
VERSION="$(git -C "$REPO_ROOT" rev-parse HEAD)"

compose_test() {
  docker compose \
    --env-file "$TEST_ENV" \
    -p "$TEST_PROJECT" \
    -f "$REPO_ROOT/compose.prod.yaml" \
    "$@"
}

compose_microsoft_test() {
  docker compose \
    --env-file "$MICROSOFT_TEST_ENV" \
    -p "$TEST_PROJECT-microsoft" \
    -f "$REPO_ROOT/compose.prod.yaml" \
    -f "$REPO_ROOT/compose.microsoft365.yaml" \
    "$@"
}

cleanup() {
  local status="$?"
  trap - EXIT

  if (( status != 0 )); then
    compose_test logs --no-color --tail=200 || true
  fi
  compose_test down --volumes --remove-orphans >/dev/null 2>&1 || true
  case "$TEST_DIR" in
    /tmp/illarin-stack.*)
      docker run --rm -v "$TEST_DIR:/test-data" alpine:3.23 \
        chown -R "$(id -u):$(id -g)" /test-data >/dev/null 2>&1 || true
      rm -rf -- "$TEST_DIR"
      ;;
  esac
  exit "$status"
}
trap cleanup EXIT

for command in docker curl; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "$command is required for the production stack test." >&2
    exit 1
  fi
done

mkdir -p \
  "$TEST_DIR/data/postgres" \
  "$TEST_DIR/data/uploads/blobs" \
  "$TEST_DIR/data/uploads/derivatives" \
  "$TEST_DIR/data/backup-work" \
  "$TEST_DIR/secrets"
chmod 0777 "$TEST_DIR/data/uploads" "$TEST_DIR/data/uploads/blobs" "$TEST_DIR/data/uploads/derivatives"
printf '%s' 'test-client-secret' >"$TEST_DIR/secrets/microsoft-365-client-secret"

{
  printf 'ILLARIN_IMAGE_REGISTRY=local\n'
  printf 'ILLARIN_VERSION=%s\n' "$VERSION"
  printf 'ILLARIN_DATA_DIR=%s/data\n' "$TEST_DIR"
  printf 'ILLARIN_SECRETS_DIR=%s/secrets\n' "$TEST_DIR"
  printf 'ILLARIN_GATEWAY_BIND=127.0.0.1\n'
  printf 'ILLARIN_GATEWAY_PORT=%s\n' "$TEST_PORT"
  printf 'NPMPLUS_NETWORK=\n'
  printf 'SITE_URL=http://localhost:%s\n' "$TEST_PORT"
  printf 'BLOG_URL=http://blog.localhost:%s\n' "$TEST_PORT"
  printf 'POSTGRES_DB=illarin\n'
  printf 'POSTGRES_USER=illarin\n'
  printf 'POSTGRES_PASSWORD=illarin-test-password\n'
  printf 'DATABASE_URL=postgres://illarin:illarin-test-password@db:5432/illarin\n'
  printf 'LINKING_HMAC_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\n'
  printf 'PUBLICATION_SECRET_KEY=BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB\n'
  printf 'SMTP_ADDR=smtp.illarin.test:25\n'
  printf 'SMTP_FROM=mail@illarin.test\n'
  printf 'UMAMI_DATABASE_PASSWORD=umami-test-password\n'
  printf 'UMAMI_ADMIN_PASSWORD=umami-test-admin-password\n'
  printf 'UMAMI_APP_SECRET=umami-test-app-secret\n'
  printf 'DD_API_KEY=00000000000000000000000000000000\n'
  printf 'BACKUPS_ENABLED=false\n'
} >"$TEST_ENV"
grep -v '^SMTP_' "$TEST_ENV" >"$MICROSOFT_TEST_ENV"
{
  printf 'MICROSOFT_365_TENANT_ID=test-tenant\n'
  printf 'MICROSOFT_365_CLIENT_ID=test-client\n'
  printf 'MICROSOFT_365_MAILBOX=mail@illarin.test\n'
} >>"$MICROSOFT_TEST_ENV"
chmod 0600 "$TEST_ENV"
chmod 0600 "$MICROSOFT_TEST_ENV"
set -a
# shellcheck disable=SC1090
source "$TEST_ENV"
set +a

docker tag illarin-api:local "local/illarin-api:$VERSION"
docker tag illarin-web:local "local/illarin-web:$VERSION"

compose_test config --quiet
if ! compose_test config | grep -Fq 'SMTP_ADDR: smtp.illarin.test:25'; then
  echo "The production stack did not pass SMTP settings to the API." >&2
  exit 1
fi
compose_microsoft_test config --quiet
compose_test up -d --wait --wait-timeout 180 db
analytics_test() {
  COMPOSE_PROJECT_NAME="$TEST_PROJECT" ILLARIN_ENV_FILE="$TEST_ENV" "$OPS_DIR/analytics.sh" "$1"
}
analytics_test prepare
compose_test up -d --wait --wait-timeout 180 umami analytics-retention
compose_test --profile tools run --rm migrate
compose_test up -d --wait --wait-timeout 240 api web gateway
compose_test ps

curl --fail --silent --show-error "http://127.0.0.1:$TEST_PORT/gateway-healthz" --output /dev/null
curl --fail --silent --show-error "http://127.0.0.1:$TEST_PORT/api/readyz" --output /dev/null
curl --fail --silent --show-error "http://127.0.0.1:$TEST_PORT/" --output /dev/null
curl --fail --silent --show-error "http://127.0.0.1:$TEST_PORT/openapi.yaml" --output /dev/null

internal_status="$(curl --silent --output /dev/null --write-out '%{http_code}' "http://127.0.0.1:$TEST_PORT/_illarin/blobs/not-public")"
if [[ "$internal_status" != "404" ]]; then
  echo "The internal blob location returned $internal_status instead of 404." >&2
  exit 1
fi

# The gateway tells the site and the blog apart by hostname, so each check
# names the hostname it speaks to and never follows a redirect.
blog_host="blog.localhost:$TEST_PORT"
expect_through_gateway() {
  local want_status="$1" host="$2" method="$3" path="$4" want_location="${5:-}" want_body="${6:-}"
  local body="$TEST_DIR/answer"
  local status redirect
  read -r status redirect < <(curl --silent --request "$method" --header "Host: $host" \
    --output "$body" --write-out '%{http_code} %{redirect_url}\n' "http://127.0.0.1:$TEST_PORT$path")
  if [[ "$status" != "$want_status" ]]; then
    echo "$method $host$path returned $status instead of $want_status." >&2
    exit 1
  fi
  if [[ -n "$want_location" && "$redirect" != "$want_location" ]]; then
    echo "$method $host$path redirected to $redirect instead of $want_location." >&2
    exit 1
  fi
  if [[ -n "$want_body" ]] && ! grep -Fq -- "$want_body" "$body"; then
    echo "$method $host$path did not answer with $want_body." >&2
    exit 1
  fi
}

expect_through_gateway 308 "127.0.0.1:$TEST_PORT" GET /blog "http://$blog_host/"
expect_through_gateway 200 "$blog_host" GET / "" "<link rel=\"canonical\" href=\"http://$blog_host"
expect_through_gateway 200 "$blog_host" GET /feed.xml "" "<rss"
expect_through_gateway 404 "$blog_host" GET /api/v1/auth/session
expect_through_gateway 404 "$blog_host" POST /api/v1/auth/sign-in
expect_through_gateway 404 "$blog_host" POST /api/v1/publication/posts
expect_through_gateway 404 "$blog_host" GET /sign-in
expect_through_gateway 404 "$blog_host" GET /admin/blog
expect_through_gateway 404 "$blog_host" GET /withdrawn

analytics_test setup
analytics_test setup
site_page="$(curl --fail --silent --show-error --header "Host: localhost:$TEST_PORT" "http://127.0.0.1:$TEST_PORT/legal/privacy")"
website_id="$(grep -m 1 -o 'data-website-id="[0-9a-f-]*"' <<<"$site_page" | cut -d '"' -f 2)"
if ! grep -Fq 'src="/stats/script.js"' <<<"$site_page" || [[ -z "$website_id" ]]; then
  echo "The site's pages do not load the analytics tracker." >&2
  exit 1
fi
expect_through_gateway 200 "localhost:$TEST_PORT" GET /stats/script.js "" "website-id"
expect_through_gateway 200 "$blog_host" GET /stats/script.js "" "website-id"
expect_through_gateway 200 "analytics.localhost:$TEST_PORT" GET /api/heartbeat
expect_through_gateway 200 "$blog_host" GET / "" 'src="/stats/script.js"'

umami_sql() {
  compose_test exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -At -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
}
send_headers="$TEST_DIR/send-headers"
curl --fail --silent --show-error --output /dev/null --dump-header "$send_headers" \
  --header "Host: $blog_host" --header 'Content-Type: application/json' \
  --header 'User-Agent: Mozilla/5.0 (X11; Linux x86_64) Firefox/140.0' \
  --data "{\"type\":\"event\",\"payload\":{\"website\":\"$website_id\",\"hostname\":\"blog.localhost\",\"url\":\"/stack-test\",\"referrer\":\"https://example.org/\",\"language\":\"en-US\",\"screen\":\"1280x720\",\"title\":\"Stack test\"}}" \
  "http://127.0.0.1:$TEST_PORT/stats/api/send"
if grep -qi '^set-cookie:' "$send_headers"; then
  echo "Counting a page view set a cookie." >&2
  exit 1
fi
if [[ "$(umami_sql <<<"SELECT count(*) FROM umami.website_event WHERE url_path = '/stack-test' AND referrer_domain = 'example.org';")" != 1 ]]; then
  echo "Umami did not record a page view sent through the gateway." >&2
  exit 1
fi
umami_sql >/dev/null <<SQL
UPDATE umami.website_event SET created_at = now() - interval '31 days' WHERE url_path = '/stack-test';
UPDATE umami.session SET created_at = now() - interval '31 days';
SQL
compose_test exec -T analytics-retention /analytics-retention.sh once
if [[ "$(umami_sql <<<"SELECT (SELECT count(*) FROM umami.website_event) + (SELECT count(*) FROM umami.session);")" != 0 ]]; then
  echo "The analytics retention job left rows older than 30 days." >&2
  exit 1
fi

expect_through_gateway 200 "127.0.0.1:$TEST_PORT" GET /developers/publication "" "Save the writing"
for page in requests writing publishing document markdown webhooks; do
  expect_through_gateway 200 "127.0.0.1:$TEST_PORT" GET "/developers/publication/$page" "" "Publication API"
done
compose_test exec -T api test -x /app/publication-authority
if [[ -n "$(compose_test exec -T web find /app -path /app/node_modules -prune -o -name '.env*' -print)" ]]; then
  echo "An environment file was copied into the web image." >&2
  exit 1
fi

credential_marker="synthetic-credential-must-not-reach-logs"
check_private_requests() {
  local host path status unavailable="${1:-false}"
  for host in "127.0.0.1:$TEST_PORT" "$blog_host"; do
    for path in \
      "/reset-password?token=$credential_marker" \
      "/verify-email?token=$credential_marker" \
      "/api/v1/auth/discord/callback?code=$credential_marker&state=$credential_marker" \
      "/link?code=$credential_marker" \
      "/api/v1/link/requests/$credential_marker" \
      "/api/v1/%6cink/requests/$credential_marker" \
      "/"; do
      curl --silent --show-error --max-time 3 --header "Host: $host" \
        --header "Referer: http://$host/%72eset-password?token=$credential_marker" \
        --output /dev/null "http://127.0.0.1:$TEST_PORT$path" || {
          status="$?"
          if [[ "$unavailable" != true || "$status" != 28 ]]; then
            return "$status"
          fi
        }
    done
  done
  compose_test logs --no-color gateway web api >"$TEST_DIR/request-logs"
  if grep -Fq "$credential_marker" "$TEST_DIR/request-logs"; then
    echo "A credential appeared in application or gateway logs." >&2
    exit 1
  fi
}

check_private_requests
compose_test stop web
check_private_requests true

echo "The isolated production stack passed."
