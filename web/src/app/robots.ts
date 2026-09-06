import type { MetadataRoute } from "next";
import { blogAddress } from "@/lib/post-link";
import { BLOG_HOME } from "@/lib/publication-metadata";
import { siteUrl } from "@/lib/site-metadata";

export const dynamic = "force-dynamic";

/** Asset pages stay crawlable on purpose. An unlisted one asks not to be indexed with a tag, and a crawler has to read the page to find it. */
export function buildRobots(origin: string): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: [
        "/api/",
        "/download/",
        "/link",
        "/settings",
        "/upload",
        "/sign-in",
        "/sign-up",
        "/verify-email",
        "/forgot-password",
        "/reset-password",
      ],
    },
    sitemap: [
      new URL("/sitemap.xml", origin).href,
      blogAddress(`${BLOG_HOME}/sitemap.xml`),
    ],
  };
}

export default function robots(): MetadataRoute.Robots {
  return buildRobots(siteUrl);
}
