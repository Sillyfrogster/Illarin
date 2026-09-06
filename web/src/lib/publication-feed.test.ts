import { expect, test } from "bun:test";
import type { PostSummary } from "@/lib/api/query";
import { publicationFeed } from "@/lib/publication-feed";

const POST: PostSummary = {
  id: "8ec131f9-96c4-437c-9cae-4db8ed28cc65",
  slug: "first-post",
  title: "Illarin keeps its own writing",
  summary: "Where announcements, releases and articles live now.",
  category: {
    id: "9d3f1c00-0000-4000-8000-000000000013",
    slug: "article",
    label: "Article",
    position: 2,
    retired: false,
  },
  byline: {
    handle: "wren",
    displayName: "Wren Ashdown",
    contactEmail: "",
    historical: false,
    positions: [],
    distinctions: [],
  },
  publishedAt: "2026-08-29T10:33:48Z",
};

test("the feed is one channel of the posts it was given", () => {
  const feed = publicationFeed([POST]);
  expect(feed).toStartWith('<?xml version="1.0" encoding="UTF-8"?>');
  expect(feed).toContain('<rss version="2.0"');
  expect(feed).toContain("<title>Illarin keeps its own writing</title>");
  expect(feed).toContain("<link>http://localhost:8000/blog/first-post</link>");
  expect(feed).toContain("<pubDate>Sat, 29 Aug 2026 10:33:48 GMT</pubDate>");
});

test("an item answers to the post rather than to the address it sits at", () => {
  expect(publicationFeed([POST])).toContain(
    '<guid isPermaLink="false">8ec131f9-96c4-437c-9cae-4db8ed28cc65</guid>',
  );
});

test("an item carries the hand-written summary and never a body", () => {
  const feed = publicationFeed([POST]);
  expect(feed).toContain(
    "<description>Where announcements, releases and articles live now.</description>",
  );
  expect(feed).not.toContain("<content");
});

test("text that would break the document is escaped", () => {
  const feed = publicationFeed([
    { ...POST, title: 'Tags, "quotes" & <angles>' },
  ]);
  expect(feed).toContain(
    "<title>Tags, &quot;quotes&quot; &amp; &lt;angles&gt;</title>",
  );
});

test("an empty publication is still a valid feed", () => {
  const feed = publicationFeed([]);
  expect(feed).toContain("<channel>");
  expect(feed).not.toContain("<item>");
});
