import type { Metadata } from "next";
import { BlogFooter } from "@/components/blog/BlogFooter";
import { BlogMasthead } from "@/components/blog/BlogMasthead";
import { fetchPostCategories } from "@/lib/api/query";
import { BLOG_TITLE, feedTypes } from "@/lib/blog-metadata";
import { BLOG_HOME } from "@/lib/blog-paths";

/** Readers get the blog as it is now, so no blog page is frozen at build time. */
export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: { default: BLOG_TITLE, template: `%s · ${BLOG_TITLE}` },
  alternates: { types: feedTypes(BLOG_HOME) },
};

export default async function BlogLayout({ children }: LayoutProps<"/blog">) {
  const categories = await fetchPostCategories();
  return (
    <>
      <a
        className="sr-only rounded-control bg-plane p-4 text-ui text-ink shadow-popover focus:not-sr-only focus:absolute focus:top-3 focus:left-3 focus:z-90"
        href="#main-content"
      >
        Skip to content
      </a>
      <BlogMasthead categories={categories} />
      <main id="main-content">{children}</main>
      <BlogFooter />
    </>
  );
}
