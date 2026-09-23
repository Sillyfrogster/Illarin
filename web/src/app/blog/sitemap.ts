import type { MetadataRoute } from "next";
import { fetchPostCategories } from "@/lib/api/query";
import { archivePath, BLOG_HOME } from "@/lib/blog-paths";
import { readBlog } from "@/lib/blog-record";
import { blogSitemap, type CrawlableArchive } from "@/lib/blog-sitemap";

export const dynamic = "force-dynamic";

const EVERY_POST = Number.POSITIVE_INFINITY;

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const [whole, categories] = await Promise.all([
    readBlog({}, EVERY_POST),
    fetchPostCategories(),
  ]);
  const narrowed = await Promise.all(
    categories.map((category) => scoped(category.slug)),
  );
  return blogSitemap(whole?.posts ?? [], [
    { address: BLOG_HOME, pages: whole?.pages ?? 0 },
    ...narrowed.filter((archive) => archive !== null),
  ]);
}

async function scoped(slug: string): Promise<CrawlableArchive | null> {
  const record = await readBlog({ category: slug }, 0);
  if (!record) return null;
  return { address: archivePath("category", slug), pages: record.pages };
}
