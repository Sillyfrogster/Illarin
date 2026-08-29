import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { cache } from "react";
import { Shell } from "@/components/layout/Shell";
import { Article } from "@/components/publication/Article";
import { fetchPublishedPost } from "@/lib/api/query";
import { pageMetadata } from "@/lib/site-metadata";

const loadPost = cache(async (slug: string) => fetchPublishedPost(slug));

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const post = await loadPost((await params).slug);
  if (!post) return { title: "Not found" };
  const metadata = pageMetadata(post.title, post.summary);
  return {
    ...metadata,
    alternates: { canonical: `/blog/${post.slug}` },
    openGraph: {
      ...metadata.openGraph,
      type: "article",
      publishedTime: post.publishedAt,
      modifiedTime: post.updatedAt,
      authors: [`@${post.byline.handle}`],
    },
  };
}

export default async function PostPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const post = await loadPost((await params).slug);
  if (!post) notFound();
  return (
    <Shell>
      <Article post={post} />
    </Shell>
  );
}
