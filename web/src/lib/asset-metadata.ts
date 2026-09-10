import type { Metadata } from "next";
import type { AssetDetail } from "./api/query";
import { assetDisplayName } from "./asset-name";
import { assetHref } from "./asset-url";
import { KIND_LABELS } from "./kinds";
import { SITE_CARD, siteOpenGraph, siteTwitter } from "./site-metadata";

export function assetMetadata(asset: AssetDetail): Metadata {
  const name = assetDisplayName(asset.name);
  const title = `${name} · ${KIND_LABELS[asset.kind]}`;
  const description = asset.blurb || `A ${asset.kind} by ${asset.creator}.`;
  const url = assetHref(asset.id, asset.name);
  const images = asset.preview
    ? [{ url: asset.preview, alt: name, width: 1200, height: 630 }]
    : [SITE_CARD];

  return {
    title,
    description,
    alternates: { canonical: url },
    robots: asset.discovery === "unlisted" ? { index: false } : undefined,
    openGraph: {
      ...siteOpenGraph(),
      type: "article",
      title,
      description,
      url,
      images,
    },
    twitter: {
      ...siteTwitter(),
      title,
      description,
      images: [asset.preview ?? SITE_CARD.url],
    },
  };
}
