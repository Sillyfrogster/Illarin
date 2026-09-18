import type { Metadata } from "next";
import type { WorkDetail } from "./api/query";
import { SITE_CARD, siteOpenGraph, siteTwitter } from "./site-metadata";
import { workDisplayName } from "./work-name";
import { TYPE_LABELS } from "./work-types";
import { workHref } from "./work-url";

export function workMetadata(work: WorkDetail): Metadata {
  const name = workDisplayName(work.name);
  const title = `${name} · ${TYPE_LABELS[work.type]}`;
  const description = work.blurb || `A ${work.type} by ${work.creator}.`;
  const url = workHref(work.id, work.name);
  const images = work.preview
    ? [{ url: work.preview, alt: name, width: 1200, height: 630 }]
    : [SITE_CARD];

  return {
    title,
    description,
    alternates: { canonical: url },
    robots: work.visibility === "unlisted" ? { index: false } : undefined,
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
      images: [work.preview ?? SITE_CARD.url],
    },
  };
}
