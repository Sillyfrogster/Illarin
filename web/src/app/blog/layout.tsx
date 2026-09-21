import type { Metadata } from "next";
import { BlogMasthead } from "@/components/blog/BlogMasthead";
import { SiteChrome } from "@/components/layout/SiteChrome";
import { fetchPostCategories } from "@/lib/api/query";
import { BLOG_TITLE, feedTypes } from "@/lib/blog-metadata";
import { BLOG_HOME } from "@/lib/blog-paths";
import { SiteProviders } from "../(site)/site-providers";

/** Readers get the blog as it is now, so no blog page is frozen at build time. */
export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: { default: BLOG_TITLE, template: `%s · ${BLOG_TITLE}` },
  alternates: { types: feedTypes(BLOG_HOME) },
};

export default async function BlogLayout({ children }: LayoutProps<"/blog">) {
  const categories = await fetchPostCategories();
  return (
    <SiteProviders>
      <SiteChrome>
        <BlogMasthead categories={categories} />
        {children}
      </SiteChrome>
    </SiteProviders>
  );
}
