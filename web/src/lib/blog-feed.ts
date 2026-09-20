import type { PostSummary } from "@/lib/api/query";
import { blogAddress, postPermalink } from "@/lib/blog-address";
import { BLOG_DESCRIPTION, BLOG_TITLE } from "@/lib/blog-metadata";
import { BLOG_HOME, feedAddresses } from "@/lib/blog-paths";
import { bylineName } from "@/lib/byline";
import { bylineProfile } from "@/lib/site-address";

export const FEED_LENGTH = 20;

export type FeedScope = {
  name: string;
  description: string;
  archive: string;
};

export const BLOG_SCOPE: FeedScope = {
  name: BLOG_TITLE,
  description: BLOG_DESCRIPTION,
  archive: BLOG_HOME,
};

export function blogFeed(scope: FeedScope, posts: PostSummary[]): string {
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

export function blogJsonFeed(scope: FeedScope, posts: PostSummary[]): string {
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

const ILLARIN_NAMESPACE = "https://illarin.com/ns/blog";

function channelName(scope: FeedScope): string {
  return scope.archive === BLOG_HOME
    ? scope.name
    : `${scope.name} · ${BLOG_TITLE}`;
}

function listed(posts: PostSummary[]): PostSummary[] {
  return posts.slice(0, FEED_LENGTH);
}

function item(post: PostSummary): string {
  const updated = post.updatedAt
    ? `\n      <atom:updated>${new Date(post.updatedAt).toISOString()}</atom:updated>`
    : "";
  return `    <item>
      <title>${escaped(post.title)}</title>
      <link>${escaped(postPermalink(post.slug))}</link>
      <guid isPermaLink="true">${escaped(postPermalink(post.originalSlug))}</guid>
      <description>${escaped(post.summary)}</description>
      <dc:creator>${escaped(bylineName(post.byline))}</dc:creator>
      <category>${escaped(post.category.label)}</category>
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
    tags: [post.category.label],
  };
}

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
