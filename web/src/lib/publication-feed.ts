import type { PostSummary } from "@/lib/api/query";
import { bylineName, bylineProfile } from "@/lib/byline";
import { blogAddress, postPermalink } from "@/lib/post-link";
import {
  BLOG_DESCRIPTION,
  BLOG_HOME,
  BLOG_TITLE,
  feedAddresses,
} from "@/lib/publication-metadata";

/** How much of the publication one feed carries before a reader has to browse. */
export const FEED_LENGTH = 20;

/** The publication an RSS or JSON feed summarizes, whole or narrowed to one archive. */
export type FeedScope = {
  name: string;
  description: string;
  archive: string;
};

export const PUBLICATION_SCOPE: FeedScope = {
  name: BLOG_TITLE,
  description: BLOG_DESCRIPTION,
  archive: BLOG_HOME,
};

/** The publication as a feed reader pulls it, with summaries and links but never a body. */
export function publicationFeed(
  scope: FeedScope,
  posts: PostSummary[],
): string {
  const items = listed(posts)
    .map((post) => item(post))
    .join("\n");
  return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:illarin="${ILLARIN_NAMESPACE}">
  <channel>
    <title>${escaped(channelName(scope))}</title>
    <link>${escaped(blogAddress(scope.archive))}</link>
    <description>${escaped(scope.description)}</description>
    <language>en-GB</language>
    <atom:link href="${escaped(blogAddress(feedAddresses(scope.archive).rss))}" rel="self" type="application/rss+xml"/>
${items}
  </channel>
</rss>
`;
}

/** The same publication for a reader that prefers JSON Feed to RSS. */
export function publicationJsonFeed(
  scope: FeedScope,
  posts: PostSummary[],
): string {
  return `${JSON.stringify(
    {
      version: "https://jsonfeed.org/version/1.1",
      title: channelName(scope),
      description: scope.description,
      language: "en-GB",
      home_page_url: blogAddress(scope.archive),
      feed_url: blogAddress(feedAddresses(scope.archive).json),
      items: listed(posts).map((post) => jsonItem(post)),
    },
    null,
    2,
  )}\n`;
}

const ILLARIN_NAMESPACE = "https://illarin.xyz/ns/publication";

function channelName(scope: FeedScope): string {
  return scope.archive === BLOG_HOME
    ? scope.name
    : `${scope.name} · ${BLOG_TITLE}`;
}

function listed(posts: PostSummary[]): PostSummary[] {
  return posts.slice(0, FEED_LENGTH);
}

function item(post: PostSummary): string {
  const release = post.releaseVersion
    ? `\n      <illarin:version>${escaped(post.releaseVersion)}</illarin:version>`
    : "";
  const app = post.app
    ? `\n      <category>${escaped(post.app.name)}</category>` +
      `\n      <illarin:app>${escaped(post.app.name)}</illarin:app>`
    : "";
  const updated = post.updatedAt
    ? `\n      <atom:updated>${new Date(post.updatedAt).toISOString()}</atom:updated>`
    : "";
  return `    <item>
      <title>${escaped(post.title)}</title>
      <link>${escaped(postPermalink(post.slug))}</link>
      <guid isPermaLink="true">${escaped(postPermalink(post.originalSlug))}</guid>
      <description>${escaped(post.summary)}</description>
      <dc:creator>${escaped(bylineName(post.byline))}</dc:creator>
      <category>${escaped(post.category.label)}</category>${app}${release}
      <pubDate>${new Date(post.publishedAt).toUTCString()}</pubDate>${updated}
    </item>`;
}

function jsonItem(post: PostSummary): Record<string, unknown> {
  const profile = bylineProfile(post.byline);
  return {
    id: postPermalink(post.originalSlug),
    url: postPermalink(post.slug),
    title: post.title,
    summary: post.summary,
    content_text: post.summary,
    date_published: new Date(post.publishedAt).toISOString(),
    ...(post.updatedAt
      ? { date_modified: new Date(post.updatedAt).toISOString() }
      : {}),
    authors: [
      {
        name: bylineName(post.byline),
        ...(profile ? { url: profile } : {}),
      },
    ],
    tags: post.app
      ? [post.category.label, post.app.name]
      : [post.category.label],
    ...(post.app || post.releaseVersion
      ? {
          _illarin: {
            ...(post.app ? { app: post.app.name } : {}),
            ...(post.releaseVersion ? { version: post.releaseVersion } : {}),
          },
        }
      : {}),
  };
}

// XML 1.0 has no way to write most control characters, so they are dropped rather than escaped.
// biome-ignore lint/suspicious/noControlCharactersInRegex: matching them is the point
const UNWRITABLE = /[\u0000-\u0008\u000B\u000C\u000E-\u001F\uFFFE\uFFFF]/g;

function escaped(said: string): string {
  return said
    .replace(UNWRITABLE, "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}
