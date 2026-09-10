import type { NextConfig } from "next";

const apiUrl = process.env.API_URL ?? "http://localhost:8080";

// An upload passes through the rewrite below and proxy.ts makes Next buffer it, and anything over this is truncated rather than refused, so it matches nginx and leaves the refusal to the API's own MAX_UPLOAD_BYTES.
const uploadBodyCeiling = "34mb";

const nextConfig: NextConfig = {
  distDir: process.env.WEB_DIST_DIR ?? ".next",
  experimental: { proxyClientMaxBodySize: uploadBodyCeiling },
  output: "standalone",
  deploymentId: process.env.ILLARIN_VERSION,
  logging: {
    incomingRequests: {
      // These URLs carry short-lived link secrets. nginx redacts them too.
      ignore: [
        /^\/link(?:\?|$)/,
        /^\/api\/v1\/link\/requests\/[^/]+/,
        /^\/api\/v1\/link\/authorizations\/[^/]+/,
      ],
    },
  },
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          {
            key: "Content-Security-Policy",
            value: "frame-ancestors 'none'",
          },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "no-referrer" },
        ],
      },
    ];
  },
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${apiUrl}/:path*`,
      },
    ];
  },
};

export default nextConfig;
