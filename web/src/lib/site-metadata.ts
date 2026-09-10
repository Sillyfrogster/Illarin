import type { Metadata } from "next";

/** The public origin, so previews, canonicals and the sitemap carry absolute URLs */
export const siteUrl = process.env.SITE_URL ?? "http://localhost:8000";

/** The blog's own public origin, which every publication address is built from */
export const blogUrl = process.env.BLOG_URL ?? siteUrl;

/** Where the server reads uploaded bytes, which is the gateway rather than the API */
export const mediaUrl = process.env.MEDIA_URL ?? siteUrl;

export const SITE_NAME = "Illarin";

export const SITE_DESCRIPTION =
  "Discover characters, lorebooks, presets, themes, and packs while keeping every creator's source file intact.";

/** The card an unfurler shows for a page with no image of its own */
export const SITE_CARD = {
  url: "/site-card.png",
  alt: "Illarin — AI roleplay, in one catalog.",
  width: 1200,
  height: 630,
  type: "image/png",
} as const;

/** A page that sets `openGraph` replaces the whole object, so every page spreads this */
export function siteOpenGraph(): NonNullable<Metadata["openGraph"]> {
  return {
    type: "website",
    siteName: SITE_NAME,
    locale: "en_GB",
    images: [SITE_CARD],
  };
}

/** The same defaults for the tags X reads */
export function siteTwitter(): NonNullable<Metadata["twitter"]> {
  return { card: "summary_large_image", images: [SITE_CARD.url] };
}

/** The tags for a page whose own title and description are all it needs */
export function pageMetadata(title: string, description: string): Metadata {
  return {
    title,
    description,
    openGraph: { ...siteOpenGraph(), title, description },
    twitter: { ...siteTwitter(), title, description },
  };
}

/**
 * What a page's metadata could read. Metadata runs before the page and outside
 * its error boundary, so a fault here has to answer as nothing and leave the
 * page itself to report it.
 */
export function readableForMetadata<T>(
  reading: Promise<T | null>,
): Promise<T | null> {
  return reading.catch(() => null);
}
