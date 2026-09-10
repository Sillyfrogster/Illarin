import { notFound } from "next/navigation";
import type { PostSummary } from "@/lib/api/query";
import {
  FEED_LENGTH,
  type FeedScope,
  PUBLICATION_SCOPE,
  publicationFeed,
  publicationJsonFeed,
} from "@/lib/publication-feed";
import { archiveDescription } from "@/lib/publication-metadata";
import { readPublication } from "@/lib/publication-record";

export type FeedForm = "rss" | "json";

const CONTENT_TYPE = {
  rss: "application/rss+xml; charset=utf-8",
  json: "application/feed+json; charset=utf-8",
} as const;

const FEED_MAX_AGE = 300;

export async function publicationFeedResponse(
  form: FeedForm,
): Promise<Response> {
  const record = await readPublication({}, FEED_LENGTH);
  return feedResponse(form, PUBLICATION_SCOPE, record?.posts ?? []);
}

export async function scopedFeedResponse(
  scope: "category" | "app",
  slug: string,
  form: FeedForm,
): Promise<Response> {
  const record = await readPublication({ [scope]: slug }, FEED_LENGTH);
  const name =
    scope === "category" ? record?.category?.label : record?.app?.name;
  if (!record || !name) notFound();
  return feedResponse(
    form,
    {
      name,
      description: archiveDescription(scope, name),
      archive: `/blog/${scope}/${slug}`,
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
    form === "rss"
      ? publicationFeed(scope, posts)
      : publicationJsonFeed(scope, posts);
  return new Response(body, {
    headers: {
      "cache-control": `public, max-age=${FEED_MAX_AGE}`,
      "content-type": CONTENT_TYPE[form],
    },
  });
}
