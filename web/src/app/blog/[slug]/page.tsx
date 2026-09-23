import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";
import { cache } from "react";
import { Article } from "@/components/blog/Article";
import { fetchPublishedPost } from "@/lib/api/query";
import { postMetadata, postStructuredData } from "@/lib/blog-metadata";
import { postPath } from "@/lib/blog-paths";
import { readableForMetadata } from "@/lib/site-metadata";

const loadPost = cache(async (slug: string) =>
  fetchPublishedPost(decodeURIComponent(slug)),
);

export async function generateMetadata({
  params,
}: PageProps<"/blog/[slug]">): Promise<Metadata> {
  const post = await readableForMetadata(loadPost((await params).slug));
  return post ? postMetadata(post) : { title: "Not found" };
}

export default async function PostPage({ params }: PageProps<"/blog/[slug]">) {
  const requested = (await params).slug;
  const post = await loadPost(requested);
  if (!post) notFound();
  if (decodeURIComponent(requested) !== post.slug) {
    permanentRedirect(postPath(post.slug));
  }
  return (
    <>
      <script type="application/ld+json">{postStructuredData(post)}</script>
      <Article post={post} />
    </>
  );
}
