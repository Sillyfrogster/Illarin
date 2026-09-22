import type { Metadata } from "next";
import type { PublicPost } from "@/lib/api/query";
import { blogAddress, postPermalink } from "@/lib/blog-address";
import { BLOG_HOME, feedAddresses, postPath } from "@/lib/blog-paths";
import { bylineName } from "@/lib/byline";
import { bylineProfile } from "@/lib/site-address";
import {
  CARD_SIZE,
  pageMetadata,
  SITE_NAME,
  siteUrl,
} from "@/lib/site-metadata";

export const BLOG_TITLE = "Illarin Blog";

export const BLOG_HEADING = "Illarin Blog";

export const BLOG_DESCRIPTION =
  "Official announcements, releases and articles from Illarin and the projects it publishes for.";

export function archiveDescription(name: string): string {
  return `Everything Illarin has published under ${name}.`;
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
  const card = linkCard(post);
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
      image: linkCard(post).url,
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

export function linkCard(post: PublicPost): {
  url: string;
  width: number;
  height: number;
  alt: string;
} {
  return {
    url: blogAddress(`${postPath(post.slug)}/card.png`),
    ...CARD_SIZE,
    alt: post.title,
  };
}
