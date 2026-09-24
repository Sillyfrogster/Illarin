import type { NextConfig } from "next";

const apiUrl = process.env.API_URL ?? "http://localhost:8080";

/** Leaves oversized-upload refusal to the API. */
const uploadBodyCeiling = "52mb";

/** Keeps account and connection credentials out of request logs, under the old paths too until their aliases go. */
const privateRequestPaths = [
  /^\/(?:connect|link)(?:\?|$)/,
  /^\/(?:verify-email|reset-password)(?:\/|\?|$)/,
  /^\/(?:api\/)?v1\/auth\/discord\/callback(?:\/|\?|$)/,
  /^\/api\/v1\/(?:connect|link)\/requests\/[^/]+/,
  /^\/api\/v1\/(?:connect|link)\/authorizations\/[^/]+/,
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
  async redirects() {
    return [
      { source: "/link", destination: "/connect", permanent: false },
      { source: "/publication", destination: "/admin/blog", permanent: true },
      { source: "/admin/blog/:id", destination: "/posts/:id", permanent: true },
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
