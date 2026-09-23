import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { BlogFront } from "@/components/blog/Archive";
import { fetchPostArchive, fetchPublishedPost } from "@/lib/api/query";
import { blogAddress } from "@/lib/blog-address";
import { BLOG_DESCRIPTION, BLOG_TITLE, feedTypes } from "@/lib/blog-metadata";
import { BLOG_HOME } from "@/lib/blog-paths";
import { postCover } from "@/lib/post-cover";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata: Metadata = {
  ...pageMetadata(BLOG_TITLE, BLOG_DESCRIPTION),
  title: { absolute: BLOG_TITLE },
  alternates: {
    canonical: blogAddress(BLOG_HOME),
    types: feedTypes(BLOG_HOME),
  },
};

export default async function BlogHomePage() {
  const archive = await fetchPostArchive({ page: 1 });
  if (!archive) notFound();
  const lead = archive.posts[0];
  const led = lead ? await fetchPublishedPost(lead.slug) : null;
  return <BlogFront archive={archive} cover={postCover(led)} />;
}
