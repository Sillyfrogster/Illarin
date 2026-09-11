import type { Metadata } from "next";
import type { PublicPost } from "@/lib/api/query";
import { bylineName, bylineProfile } from "@/lib/byline";
import { blogAddress, postPermalink } from "@/lib/post-link";
import { pageMetadata, SITE_NAME, siteUrl } from "@/lib/site-metadata";

export const BLOG_HOME = "/blog";

export const CARD_SIZE = { width: 1200, height: 630 } as const;

export const ILLARIN_APP = "illarin";

export const BLOG_TITLE = "Illarin Blog";

export const BLOG_HEADING = "Illarin Blog";

export const BLOG_DESCRIPTION =
  "Official announcements, releases and articles from Illarin and the projects it publishes for.";

export function feedAddresses(archive: string): { rss: string; json: string } {
  return { rss: `${archive}/feed.xml`, json: `${archive}/feed.json` };
}

export const PUBLICATION_FEEDS = feedAddresses(BLOG_HOME);

export function archiveDescription(
  scope: "category" | "app",
  name: string,
): string {
  return scope === "category"
    ? `Everything Illarin has published under ${name}.`
    : `Everything Illarin has published about ${name}.`;
}

export function archivePage(paging: string[] | undefined): number | null {
  if (!paging || paging.length === 0) return 1;
  if (paging.length !== 2 || paging[0] !== "page") return null;
  if (!/^[1-9][0-9]*$/.test(paging[1])) return null;
  return Number(paging[1]);
}

export function pageAddress(address: string, page: number): string {
  return page === 1 ? address : `${address}/page/${page}`;
}

export function blogMetadata(
  name: string,
  description: string,
  canonical: string,
  archive: string,
): Metadata {
  return {
    ...pageMetadata(`${name} · ${BLOG_TITLE}`, description),
    title: name,
    alternates: {
      canonical: blogAddress(canonical),
      types: feedTypes(archive),
    },
  };
}

export function feedTypes(archive: string): Record<string, string> {
  const feeds = feedAddresses(archive);
  return {
    "application/rss+xml": blogAddress(feeds.rss),
    "application/feed+json": blogAddress(feeds.json),
  };
}

export function postMetadata(post: PublicPost): Metadata {
  const canonical = postPermalink(post.slug);
  const card = socialCard(post);
  return {
    title: post.title,
    description: post.summary,
    alternates: { canonical, types: feedTypes(BLOG_HOME) },
    openGraph: {
      type: "article",
      siteName: BLOG_TITLE,
      locale: "en_GB",
      title: post.title,
      description: post.summary,
      url: canonical,
      publishedTime: post.publishedAt,
      modifiedTime: post.updatedAt,
      authors: [bylineProfile(post.byline) ?? bylineName(post.byline)],
      images: [card],
    },
    twitter: {
      card: "summary_large_image",
      title: post.title,
      description: post.summary,
      images: [card.url],
    },
  };
}

export function postStructuredData(post: PublicPost): string {
  const profile = bylineProfile(post.byline);
  return inertInAScript(
    JSON.stringify({
      "@context": "https://schema.org",
      "@type": "Article",
      headline: post.title,
      description: post.summary,
      url: postPermalink(post.slug),
      mainEntityOfPage: postPermalink(post.slug),
      image: socialCard(post).url,
      datePublished: post.publishedAt,
      ...(post.updatedAt ? { dateModified: post.updatedAt } : {}),
      author: {
        "@type": "Person",
        name: bylineName(post.byline),
        ...(profile ? { url: profile } : {}),
      },
      publisher: { "@type": "Organization", name: SITE_NAME, url: siteUrl },
    }),
  );
}

const CLOSES_A_SCRIPT = /[<>&\u2028\u2029]/g;

function inertInAScript(json: string): string {
  return json.replace(
    CLOSES_A_SCRIPT,
    (letter) => `\\u${letter.charCodeAt(0).toString(16).padStart(4, "0")}`,
  );
}

export function socialCard(post: PublicPost): {
  url: string;
  width: number;
  height: number;
  alt: string;
} {
  const override = post.socialImage;
  return {
    url: override
      ? blogAddress(override.url)
      : blogAddress(`${BLOG_HOME}/${encodeURI(post.slug)}/card.png`),
    width: override?.width ?? CARD_SIZE.width,
    height: override?.height ?? CARD_SIZE.height,
    alt: post.title,
  };
}
