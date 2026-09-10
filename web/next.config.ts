import type { NextConfig } from "next";

const apiUrl = process.env.API_URL ?? "http://localhost:8080";

/** Leaves oversized-upload refusal to the API. */
const uploadBodyCeiling = "34mb";

/** Keeps link credentials out of request logs. */
const privateRequestPaths = [
  /^\/link(?:\?|$)/,
  /^\/api\/v1\/link\/requests\/[^/]+/,
  /^\/api\/v1\/link\/authorizations\/[^/]+/,
];

const nextConfig: NextConfig = {
  distDir: process.env.WEB_DIST_DIR ?? ".next",
  experimental: { proxyClientMaxBodySize: uploadBodyCeiling },
  output: "standalone",
  deploymentId: process.env.ILLARIN_VERSION,
  logging: {
    incomingRequests: {
      ignore: privateRequestPaths,
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
