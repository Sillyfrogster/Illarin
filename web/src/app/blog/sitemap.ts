import type { MetadataRoute } from "next";
import { fetchPostApps, fetchPostCategories } from "@/lib/api/query";
import { BLOG_HOME } from "@/lib/publication-metadata";
import { readPublication } from "@/lib/publication-record";
import {
  type CrawlableArchive,
  publicationSitemap,
} from "@/lib/publication-sitemap";

export const dynamic = "force-dynamic";

/** Every published post, however long the publication has run. */
const EVERY_POST = Number.POSITIVE_INFINITY;

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const [whole, categories, apps] = await Promise.all([
    readPublication({}, EVERY_POST),
    fetchPostCategories(),
    fetchPostApps(),
  ]);
  const narrowed = await Promise.all([
    ...categories.map((category) => scoped("category", category.slug)),
    ...apps.map((app) => scoped("app", app.slug)),
  ]);
  return publicationSitemap(whole?.posts ?? [], [
    { address: BLOG_HOME, pages: whole?.pages ?? 0 },
    ...narrowed.filter((archive) => archive !== null),
  ]);
}

async function scoped(
  scope: "category" | "app",
  slug: string,
): Promise<CrawlableArchive | null> {
  const record = await readPublication({ [scope]: slug }, 0);
  if (!record) return null;
  return { address: `${BLOG_HOME}/${scope}/${slug}`, pages: record.pages };
}
