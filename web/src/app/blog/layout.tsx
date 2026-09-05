import type { Metadata } from "next";
import { BlogFooter } from "@/components/publication/BlogFooter";
import { BlogMasthead } from "@/components/publication/BlogMasthead";
import { fetchPostCategories } from "@/lib/api/query";
import { FEED_ADDRESS } from "@/lib/publication-feed";
import { BLOG_TITLE } from "@/lib/publication-metadata";

export const metadata: Metadata = {
  title: { default: BLOG_TITLE, template: `%s · ${BLOG_TITLE}` },
  alternates: { types: { "application/rss+xml": FEED_ADDRESS } },
};

export default async function BlogLayout({ children }: LayoutProps<"/blog">) {
  const categories = await fetchPostCategories();
  return (
    <>
      <BlogMasthead categories={categories} />
      <main>{children}</main>
      <BlogFooter />
    </>
  );
}
