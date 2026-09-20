import { notFound } from "next/navigation";
import type { PostSummary } from "@/lib/api/query";
import {
  BLOG_SCOPE,
  blogFeed,
  blogJsonFeed,
  FEED_LENGTH,
  type FeedScope,
} from "@/lib/blog-feed";
import { archiveDescription } from "@/lib/blog-metadata";
import { archivePath } from "@/lib/blog-paths";
import { readBlog } from "@/lib/blog-record";

export type FeedForm = "rss" | "json";

const CONTENT_TYPE = {
  rss: "application/rss+xml; charset=utf-8",
  json: "application/feed+json; charset=utf-8",
} as const;

const FEED_MAX_AGE = 300;

export async function blogFeedResponse(form: FeedForm): Promise<Response> {
  const record = await readBlog({}, FEED_LENGTH);
  return feedResponse(form, BLOG_SCOPE, record?.posts ?? []);
}

export async function categoryFeedResponse(
  slug: string,
  form: FeedForm,
): Promise<Response> {
  const record = await readBlog({ category: slug }, FEED_LENGTH);
  const name = record?.category?.label;
  if (!record || !name) notFound();
  return feedResponse(
    form,
    {
      name,
      description: archiveDescription(name),
      archive: archivePath("category", slug),
    },
    record.posts,
  );
}

function feedResponse(
  form: FeedForm,
  scope: FeedScope,
  posts: PostSummary[],
): Response {
  const body =
    form === "rss" ? blogFeed(scope, posts) : blogJsonFeed(scope, posts);
  return new Response(body, {
    headers: {
      "cache-control": `public, max-age=${FEED_MAX_AGE}`,
      "content-type": CONTENT_TYPE[form],
    },
  });
}
