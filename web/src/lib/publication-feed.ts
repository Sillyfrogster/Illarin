import type { PostSummary } from "@/lib/api/query";
import { postPermalink } from "@/lib/post-link";
import { BLOG_DESCRIPTION, BLOG_TITLE } from "@/lib/publication-metadata";
import { siteUrl } from "@/lib/site-metadata";

/** The address the feed answers at, which it also names inside itself. */
export const FEED_ADDRESS = "/blog/feed.xml";

/** The publication as a feed reader pulls it, with summaries and links but never a body. */
export function publicationFeed(posts: PostSummary[]): string {
  const home = new URL("/blog", siteUrl).href;
  const self = new URL(FEED_ADDRESS, siteUrl).href;
  const items = posts.map((post) => item(post)).join("\n");
  return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>${escaped(BLOG_TITLE)}</title>
    <link>${escaped(home)}</link>
    <description>${escaped(BLOG_DESCRIPTION)}</description>
    <language>en-GB</language>
    <atom:link href="${escaped(self)}" rel="self" type="application/rss+xml"/>
${items}
  </channel>
</rss>
`;
}

function item(post: PostSummary): string {
  const address = postPermalink(post.slug);
  return `    <item>
      <title>${escaped(post.title)}</title>
      <link>${escaped(address)}</link>
      <guid isPermaLink="false">${escaped(post.id)}</guid>
      <description>${escaped(post.summary)}</description>
      <category>${escaped(post.category.label)}</category>
      <pubDate>${new Date(post.publishedAt).toUTCString()}</pubDate>
    </item>`;
}

function escaped(said: string): string {
  return said
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}
