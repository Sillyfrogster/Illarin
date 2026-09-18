import type { MetadataRoute } from "next";
import {
  type BrowsePage,
  fetchWorks,
  type WorkListParams,
} from "@/lib/api/query";
import { siteUrl } from "@/lib/site-metadata";
import { workHref } from "@/lib/work-url";

export const dynamic = "force-dynamic";

type ListSitemapWorks = (
  params: WorkListParams,
) => Promise<Pick<BrowsePage, "items" | "nextCursor">>;

export async function buildSitemap(
  listWorks: ListSitemapWorks,
): Promise<MetadataRoute.Sitemap> {
  const entries: MetadataRoute.Sitemap = [
    { url: new URL("/", siteUrl).href },
    { url: new URL("/browse", siteUrl).href },
  ];
  let cursor: BrowsePage["nextCursor"];

  do {
    const page = await listWorks({
      limit: 24,
      nsfw: "shown",
      before: cursor?.before,
      beforeId: cursor?.beforeId,
    });
    for (const work of page.items) {
      entries.push({
        url: new URL(workHref(work.id, work.name), siteUrl).href,
      });
    }
    cursor = page.nextCursor;
  } while (cursor);

  return entries;
}

export default function sitemap(): Promise<MetadataRoute.Sitemap> {
  return buildSitemap(fetchWorks);
}
