import type { Metadata } from "next";
import type { WorkDetail } from "./api/query";
import { CARD_SIZE, siteOpenGraph, siteTwitter } from "./site-metadata";
import { workDisplayName } from "./work-name";
import { TYPE_LABELS } from "./work-types";
import { workHref } from "./work-url";

export function workMetadata(work: WorkDetail): Metadata {
  if (work.lifecycle === "draft") {
    return { title: "Draft", robots: { index: false, follow: false } };
  }
  const name = workDisplayName(work.name);
  const title = `${name} · ${TYPE_LABELS[work.type]}`;
  const description = work.blurb || `A ${work.type} by ${work.creator}.`;
  const url = workHref(work.id, work.name);
  const card = `/a/${work.id}/card.png`;

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
      images: [{ url: card, alt: name, ...CARD_SIZE }],
    },
    twitter: {
      ...siteTwitter(),
      title,
      description,
      images: [card],
    },
  };
}
