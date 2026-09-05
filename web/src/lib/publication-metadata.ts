import type { Metadata } from "next";
import { pageMetadata } from "@/lib/site-metadata";

export const BLOG_TITLE = "Illarin Blog";

export const BLOG_DESCRIPTION =
  "Official announcements, releases and articles from Illarin and the projects it publishes for.";

/** Reads the page number out of an archive address, refusing anything but /page/<number>. */
export function archivePage(paging: string[] | undefined): number | null {
  if (!paging || paging.length === 0) return 1;
  if (paging.length !== 2 || paging[0] !== "page") return null;
  if (!/^[1-9][0-9]*$/.test(paging[1])) return null;
  return Number(paging[1]);
}

/** A short tab title under the publication's template, and the full name in a link preview. */
export function blogMetadata(
  name: string,
  description: string,
  canonical: string,
): Metadata {
  return {
    ...pageMetadata(`${name} · ${BLOG_TITLE}`, description),
    title: name,
    alternates: { canonical },
  };
}
