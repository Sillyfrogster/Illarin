import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";
import { cache } from "react";
import { Article } from "@/components/publication/Article";
import { fetchPublishedPost } from "@/lib/api/query";
import { pageMetadata } from "@/lib/site-metadata";

const loadPost = cache(async (slug: string) =>
  fetchPublishedPost(decodeURIComponent(slug)),
);

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
  const requested = (await params).slug;
  const post = await loadPost(requested);
  if (!post) notFound();
  if (decodeURIComponent(requested) !== post.slug) {
    permanentRedirect(`/blog/${post.slug}`);
  }
  return <Article post={post} />;
}
