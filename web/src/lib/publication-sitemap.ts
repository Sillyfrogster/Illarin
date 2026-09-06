import type { MetadataRoute } from "next";
import type { PostSummary } from "@/lib/api/query";
import { blogAddress, postPermalink } from "@/lib/post-link";
import { pageAddress } from "@/lib/publication-metadata";

/** One browsable archive and how many numbered pages it runs to. */
export type CrawlableArchive = {
  address: string;
  pages: number;
};

/** Every address a crawler may follow into the publication, and nothing that is not public. */
export function publicationSitemap(
  posts: PostSummary[],
  archives: CrawlableArchive[],
): MetadataRoute.Sitemap {
  const entries: MetadataRoute.Sitemap = [];
  for (const archive of archives) {
    for (let page = 1; page <= Math.max(archive.pages, 1); page += 1) {
      entries.push({ url: blogAddress(pageAddress(archive.address, page)) });
    }
  }
  for (const post of posts) {
    entries.push({
      url: postPermalink(post.slug),
      lastModified: new Date(post.updatedAt ?? post.publishedAt),
    });
  }
  return entries;
}
