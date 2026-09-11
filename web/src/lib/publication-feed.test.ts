import { expect, test } from "bun:test";
import type { PostSummary } from "@/lib/api/query";
import {
  PUBLICATION_SCOPE,
  publicationFeed,
  publicationJsonFeed,
} from "@/lib/publication-feed";

const POST: PostSummary = {
  id: "8ec131f9-96c4-437c-9cae-4db8ed28cc65",
  slug: "first-post",
  originalSlug: "first-post",
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

const RELEASE: PostSummary = {
  ...POST,
  id: "1b0b6d02-4a9e-4b90-9a35-2ba4f2d5f0a1",
  slug: "lumiverse-2-0",
  originalSlug: "lumiverse-2-0",
  title: "Lumiverse 2.0 is out",
  category: {
    id: "9d3f1c00-0000-4000-8000-000000000011",
    slug: "release",
    label: "Release",
    position: 0,
    retired: false,
  },
  app: {
    id: "3a0f0c00-0000-4000-8000-000000000001",
    slug: "lumiverse",
    name: "Lumiverse",
    home: "https://lumiverse.example",
    position: 0,
    retired: false,
    destinations: [],
  },
  releaseVersion: "2.0",
  publishedAt: "2026-08-30T08:00:00Z",
  updatedAt: "2026-09-02T11:15:00Z",
};

const CATEGORY_SCOPE = {
  name: "Release",
  description: "Everything Illarin has published under Release.",
  archive: "/category/release",
};

test("the channel names the publication and the address it answers at", () => {
  const feed = publicationFeed(PUBLICATION_SCOPE, [POST]);
  expect(feed).toStartWith('<?xml version="1.0" encoding="UTF-8"?>');
  expect(feed).toContain('<rss version="2.0"');
  expect(feed).toContain("<title>Illarin Blog</title>");
  expect(feed).toContain("<link>http://blog.localhost:8000/</link>");
  expect(feed).toContain(
    '<atom:link href="http://blog.localhost:8000/feed.xml" rel="self"',
  );
});

test("a scoped channel names its own archive and feed", () => {
  const feed = publicationFeed(CATEGORY_SCOPE, [RELEASE]);
  expect(feed).toContain("<title>Release · Illarin Blog</title>");
  expect(feed).toContain(
    "<link>http://blog.localhost:8000/category/release</link>",
  );
  expect(feed).toContain(
    '<atom:link href="http://blog.localhost:8000/category/release/feed.xml" rel="self"',
  );
});

test("an item carries the summary, attribution, category and public dates", () => {
  const feed = publicationFeed(PUBLICATION_SCOPE, [RELEASE]);
  expect(feed).toContain("<title>Lumiverse 2.0 is out</title>");
  expect(feed).toContain(
    "<link>http://blog.localhost:8000/lumiverse-2-0</link>",
  );
  expect(feed).toContain(
    "<description>Where announcements, releases and articles live now.</description>",
  );
  expect(feed).toContain("<dc:creator>Wren Ashdown</dc:creator>");
  expect(feed).toContain("<category>Release</category>");
  expect(feed).toContain("<category>Lumiverse</category>");
  expect(feed).toContain("<pubDate>Sun, 30 Aug 2026 08:00:00 GMT</pubDate>");
  expect(feed).toContain(
    "<atom:updated>2026-09-02T11:15:00.000Z</atom:updated>",
  );
});

test("an item carries the version a release post names", () => {
  expect(publicationFeed(PUBLICATION_SCOPE, [RELEASE])).toContain(
    "<illarin:version>2.0</illarin:version>",
  );
  expect(publicationFeed(PUBLICATION_SCOPE, [POST])).not.toContain(
    "<illarin:version>",
  );
});

test("a post with no update carries no updated date", () => {
  expect(publicationFeed(PUBLICATION_SCOPE, [POST])).not.toContain(
    "<atom:updated>",
  );
});

test("an item answers to the address the post first published under", () => {
  const corrected: PostSummary = { ...POST, slug: "corrected-address" };
  const feed = publicationFeed(PUBLICATION_SCOPE, [corrected]);
  expect(feed).toContain(
    '<guid isPermaLink="true">http://blog.localhost:8000/first-post</guid>',
  );
  expect(feed).toContain(
    "<link>http://blog.localhost:8000/corrected-address</link>",
  );
});

test("a feed never carries the body of a post", () => {
  const feed = publicationFeed(PUBLICATION_SCOPE, [POST, RELEASE]);
  expect(feed).not.toContain("<content");
  expect(feed).not.toContain("encoded");
});

test("text that would break the document is escaped", () => {
  expect(
    publicationFeed(PUBLICATION_SCOPE, [
      { ...POST, title: 'Tags, "quotes" & <angles>' },
    ]),
  ).toContain("<title>Tags, &quot;quotes&quot; &amp; &lt;angles&gt;</title>");
});

test("an empty publication is still a valid feed", () => {
  const feed = publicationFeed(PUBLICATION_SCOPE, []);
  expect(feed).toContain("<channel>");
  expect(feed).not.toContain("<item>");
});

test("the json feed names itself, its home and the posts it lists", () => {
  const feed = JSON.parse(publicationJsonFeed(PUBLICATION_SCOPE, [RELEASE]));
  expect(feed.version).toBe("https://jsonfeed.org/version/1.1");
  expect(feed.title).toBe("Illarin Blog");
  expect(feed.home_page_url).toBe("http://blog.localhost:8000/");
  expect(feed.feed_url).toBe("http://blog.localhost:8000/feed.json");
  expect(feed.items).toHaveLength(1);

  const [item] = feed.items;
  expect(item.id).toBe("http://blog.localhost:8000/lumiverse-2-0");
  expect(item.url).toBe("http://blog.localhost:8000/lumiverse-2-0");
  expect(item.title).toBe("Lumiverse 2.0 is out");
  expect(item.summary).toBe(
    "Where announcements, releases and articles live now.",
  );
  expect(item.content_text).toBe(item.summary);
  expect(item.date_published).toBe("2026-08-30T08:00:00.000Z");
  expect(item.date_modified).toBe("2026-09-02T11:15:00.000Z");
  expect(item.tags).toEqual(["Release", "Lumiverse"]);
  expect(item._illarin).toEqual({ app: "Lumiverse", version: "2.0" });
});

test("a json item names its author and sends a reader to the main origin", () => {
  const [item] = JSON.parse(
    publicationJsonFeed(PUBLICATION_SCOPE, [POST]),
  ).items;
  expect(item.authors).toEqual([
    { name: "Wren Ashdown", url: "http://localhost:8000/@wren" },
  ]);
});

test("a byline with no account behind it links nowhere", () => {
  const [item] = JSON.parse(
    publicationJsonFeed(PUBLICATION_SCOPE, [
      { ...POST, byline: { ...POST.byline, historical: true } },
    ]),
  ).items;
  expect(item.authors).toEqual([{ name: "Wren Ashdown" }]);
});

test("a json item answers to the address the post first published under", () => {
  const [item] = JSON.parse(
    publicationJsonFeed(PUBLICATION_SCOPE, [
      { ...POST, slug: "corrected-address" },
    ]),
  ).items;
  expect(item.id).toBe("http://blog.localhost:8000/first-post");
  expect(item.url).toBe("http://blog.localhost:8000/corrected-address");
});

test("an empty publication is still a valid json feed", () => {
  const feed = JSON.parse(publicationJsonFeed(PUBLICATION_SCOPE, []));
  expect(feed.items).toEqual([]);
});

test("a character XML cannot carry is dropped rather than written", () => {
  const feed = publicationFeed(PUBLICATION_SCOPE, [
    { ...POST, title: `A title with a ${String.fromCharCode(7)} bell in it` },
  ]);
  expect(feed).toContain("<title>A title with a  bell in it</title>");
  expect(feed).not.toContain(String.fromCharCode(7));
});
