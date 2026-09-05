import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { PublicationFront } from "@/components/publication/Archive";
import { fetchPostArchive } from "@/lib/api/query";
import { BLOG_DESCRIPTION, BLOG_TITLE } from "@/lib/publication-metadata";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata: Metadata = {
  ...pageMetadata(BLOG_TITLE, BLOG_DESCRIPTION),
  title: { absolute: BLOG_TITLE },
  alternates: { canonical: "/blog" },
};

export default async function BlogHomePage() {
  const archive = await fetchPostArchive({ page: 1 });
  if (!archive) notFound();
  return <PublicationFront archive={archive} />;
}
