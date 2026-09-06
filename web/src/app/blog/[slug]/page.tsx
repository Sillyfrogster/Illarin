import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";
import { cache } from "react";
import { Article } from "@/components/publication/Article";
import { fetchPublishedPost } from "@/lib/api/query";
import {
  BLOG_HOME,
  postMetadata,
  postStructuredData,
} from "@/lib/publication-metadata";

const loadPost = cache(async (slug: string) =>
  fetchPublishedPost(decodeURIComponent(slug)),
);

export async function generateMetadata({
  params,
}: PageProps<"/blog/[slug]">): Promise<Metadata> {
  const post = await loadPost((await params).slug);
  return post ? postMetadata(post) : { title: "Not found" };
}

export default async function PostPage({ params }: PageProps<"/blog/[slug]">) {
  const requested = (await params).slug;
  const post = await loadPost(requested);
  if (!post) notFound();
  if (decodeURIComponent(requested) !== post.slug) {
    permanentRedirect(`${BLOG_HOME}/${post.slug}`);
  }
  return (
    <>
      <script
        // biome-ignore lint/security/noDangerouslySetInnerHtml: the article data is composed from the validated post
        dangerouslySetInnerHTML={{ __html: postStructuredData(post) }}
        type="application/ld+json"
      />
      <Article post={post} />
    </>
  );
}
