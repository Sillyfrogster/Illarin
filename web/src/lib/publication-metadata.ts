import type { Metadata } from "next";
import { pageMetadata } from "@/lib/site-metadata";

/** Where the publication answers, and what every archive address is built beneath */
export const BLOG_HOME = "/blog";

export const BLOG_TITLE = "Illarin Blog";

export const BLOG_DESCRIPTION =
  "Official announcements, releases and articles from Illarin and the projects it publishes for.";

/** Where the two feeds answer beneath the archive they summarize. */
export function feedAddresses(archive: string): { rss: string; json: string } {
  return { rss: `${archive}/feed.xml`, json: `${archive}/feed.json` };
}

export const PUBLICATION_FEEDS = feedAddresses(BLOG_HOME);

/** What a narrowed archive holds, said the same way on its page and in its feed. */
export function archiveDescription(
  scope: "category" | "app",
  name: string,
): string {
  return scope === "category"
    ? `Everything Illarin has published under ${name}.`
    : `Everything Illarin has published about ${name}.`;
}

/** Reads the page number out of an archive address, refusing anything but /page/<number>. */
export function archivePage(paging: string[] | undefined): number | null {
  if (!paging || paging.length === 0) return 1;
  if (paging.length !== 2 || paging[0] !== "page") return null;
  if (!/^[1-9][0-9]*$/.test(paging[1])) return null;
  return Number(paging[1]);
}

/** Page one of any archive lives at the archive's own address, not under /page/1. */
export function pageAddress(address: string, page: number): string {
  return page === 1 ? address : `${address}/page/${page}`;
}

/** A short tab title under the publication's template, and the full name in a link preview. */
export function blogMetadata(
  name: string,
  description: string,
  canonical: string,
  archive: string,
): Metadata {
  return {
    ...pageMetadata(`${name} · ${BLOG_TITLE}`, description),
    title: name,
    alternates: { canonical, types: feedTypes(archive) },
  };
}

/** The feeds a reader can subscribe to from the archive they are looking at. */
export function feedTypes(archive: string): Record<string, string> {
  const feeds = feedAddresses(archive);
  return {
    "application/rss+xml": feeds.rss,
    "application/feed+json": feeds.json,
  };
}
