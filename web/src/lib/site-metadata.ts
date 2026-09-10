import type { Metadata } from "next";

export const siteUrl = process.env.SITE_URL ?? "http://localhost:8000";

export const blogUrl = process.env.BLOG_URL ?? siteUrl;

export const mediaUrl = process.env.MEDIA_URL ?? siteUrl;

export const SITE_NAME = "Illarin";

export const SITE_DESCRIPTION =
  "Discover characters, lorebooks, presets, themes, and packs while keeping every creator's source file intact.";

export const SITE_CARD = {
  url: "/site-card.png",
  alt: "Illarin — AI roleplay, in one catalog.",
  width: 1200,
  height: 630,
  type: "image/png",
} as const;

export function siteOpenGraph(): NonNullable<Metadata["openGraph"]> {
  return {
    type: "website",
    siteName: SITE_NAME,
    locale: "en_GB",
    images: [SITE_CARD],
  };
}

export function siteTwitter(): NonNullable<Metadata["twitter"]> {
  return { card: "summary_large_image", images: [SITE_CARD.url] };
}

export function pageMetadata(title: string, description: string): Metadata {
  return {
    title,
    description,
    openGraph: { ...siteOpenGraph(), title, description },
    twitter: { ...siteTwitter(), title, description },
  };
}

export function readableForMetadata<T>(
  reading: Promise<T | null>,
): Promise<T | null> {
  return reading.catch(() => null);
}
