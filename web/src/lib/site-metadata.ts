import type { Metadata } from "next";

export const siteUrl = process.env.SITE_URL ?? "http://localhost:8000";

export const mediaUrl = process.env.MEDIA_URL ?? siteUrl;

export const SITE_NAME = "Illarin";

export const CARD_SIZE = { width: 1200, height: 630 } as const;

export const SITE_DESCRIPTION =
  "Characters, lorebooks, presets, themes and extensions for your roleplay app";

export const SITE_CARD = {
  url: "/site-card.jpg",
  alt: "Illarin",
  width: 1200,
  height: 630,
  type: "image/jpeg",
} as const;

export function siteOpenGraph(): NonNullable<Metadata["openGraph"]> {
  return {
    type: "website",
    siteName: SITE_NAME,
    locale: "en_US",
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
