#!/usr/bin/env bash

set -euo pipefail

OPS_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$OPS_DIR/lib.sh"

require_release

site_host="$(host_of "${SITE_URL:?Set SITE_URL}")"
blog_host="blog.$site_host"
blog_root="${SITE_URL%/}/blog"

compose exec -T gateway wget -q -T 5 -O /dev/null http://127.0.0.1:8080/gateway-healthz
compose exec -T gateway wget -q -T 10 -O /dev/null http://127.0.0.1:8080/api/readyz
compose exec -T gateway wget -q -T 15 -O /dev/null http://127.0.0.1:8080/

# Every check names the hostname it speaks to and reads the answer without
# following redirects, including redirects from the old blog hostname.
through_gateway() {
  compose exec -T \
    -e "SMOKE_HOST=$1" -e "SMOKE_METHOD=$2" -e "SMOKE_PATH=$3" \
    -e "SMOKE_STATUS=$4" -e "SMOKE_LOCATION=${5:-}" -e "SMOKE_BODY=${6:-}" \
    web node -e '
const http = require("node:http");
const want = process.env;
const request = http.request(
  { host: "gateway", port: 8080, method: want.SMOKE_METHOD, path: want.SMOKE_PATH, headers: { Host: want.SMOKE_HOST } },
  (response) => {
    let body = "";
    response.setEncoding("utf8");
    response.on("data", (piece) => { if (body.length < 1 << 20) body += piece; });
    response.on("end", () => {
      const faults = [];
      if (String(response.statusCode) !== want.SMOKE_STATUS) faults.push(`status ${response.statusCode}, want ${want.SMOKE_STATUS}`);
      if (want.SMOKE_LOCATION && response.headers.location !== want.SMOKE_LOCATION) faults.push(`location ${response.headers.location}, want ${want.SMOKE_LOCATION}`);
      if (want.SMOKE_BODY && !body.includes(want.SMOKE_BODY)) faults.push(`body lacks ${want.SMOKE_BODY}`);
      if (faults.length > 0) {
        console.error(`${want.SMOKE_METHOD} ${want.SMOKE_HOST}${want.SMOKE_PATH}: ${faults.join("; ")}`);
        process.exit(1);
      }
    });
  },
);
request.on("error", (error) => { console.error(error.message); process.exit(1); });
request.setTimeout(15000, () => request.destroy(new Error("timed out")));
request.end();
'
}

through_gateway "$site_host" GET / 200
through_gateway "$site_host" GET /stats/script.js 200 "" "website-id"
through_gateway "analytics.$site_host" GET /api/heartbeat 200
through_gateway "$site_host" GET /blog 200 "" "<link rel=\"canonical\" href=\"$blog_root"
through_gateway "$site_host" GET /blog/feed.xml 200 "" "<rss"
through_gateway "$blog_host" GET / 308 "$blog_root"
through_gateway "$blog_host" GET /a-post 308 "$blog_root/a-post"
through_gateway "illarin.xyz" GET /a/1 308 "https://illarin.com/a/1"
through_gateway "blog.illarin.xyz" GET /a-post 308 "https://illarin.com/blog/a-post"

if ! compose ps --status running --services | grep -qx analytics-retention; then
  echo "The nightly analytics retention job is not running." >&2
  exit 1
fi

echo "Illarin passed its gateway, API, site, blog redirect and analytics smoke checks."
