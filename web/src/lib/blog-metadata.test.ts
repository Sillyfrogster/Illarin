import { expect, test } from "bun:test";
import type { PublicPost } from "@/lib/api/query";
import {
  archiveDescription,
  blogMetadata,
  postMetadata,
  postStructuredData,
} from "./blog-metadata";

const POST: PublicPost = {
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
  body: { version: 2, content: [] },
  media: [],
  related: [],
  byline: {
    handle: "wren",
    displayName: "Wren Ashdown",
    contactEmail: "",
    historical: false,
  },
  publishedAt: "2026-08-29T10:33:48Z",
};

const OVERRIDE = {
  id: "5d31391b-cf7e-478f-814b-c6b28639e5a3",
  postId: POST.id,
  purpose: "link_card" as const,
  url: "/media/5d31391b-cf7e-478f-814b-c6b28639e5a3/og/1",
  thumbUrl: "/media/5d31391b-cf7e-478f-814b-c6b28639e5a3/grid/1",
  width: 1200,
  height: 630,
};

test("a post is canonical at its own address on the blog origin", () => {
  const metadata = postMetadata(POST);
  expect(metadata.alternates?.canonical).toBe(
    "http://blog.localhost:8000/first-post",
  );
  expect(metadata.openGraph?.url).toBe("http://blog.localhost:8000/first-post");
});

test("a preview carries the title, the hand-written summary and the dates stored", () => {
  const metadata = postMetadata(POST);
  expect(metadata.title).toBe("Illarin keeps its own writing");
  expect(metadata.description).toBe(
    "Where announcements, releases and articles live now.",
  );
  const graph = metadata.openGraph as Record<string, unknown>;
  expect(graph.type).toBe("article");
  expect(graph.publishedTime).toBe("2026-08-29T10:33:48Z");
  expect(graph.modifiedTime).toBeUndefined();
  expect(graph.authors).toEqual(["http://localhost:8000/@wren"]);
});

test("an updated post says when it was last changed", () => {
  const graph = postMetadata({ ...POST, updatedAt: "2026-09-02T11:15:00Z" })
    .openGraph as Record<string, unknown>;
  expect(graph.modifiedTime).toBe("2026-09-02T11:15:00Z");
});

test("a byline with no account behind it is named but not linked", () => {
  const graph = postMetadata({
    ...POST,
    byline: { ...POST.byline, historical: true },
  }).openGraph as Record<string, unknown>;
  expect(graph.authors).toEqual(["Wren Ashdown"]);
});

test("a post with no override gets the card Illarin composes for it", () => {
  const metadata = postMetadata(POST);
  expect(metadata.openGraph?.images).toEqual([
    {
      url: "http://blog.localhost:8000/first-post/card.png",
      width: 1200,
      height: 630,
      alt: "Illarin keeps its own writing",
    },
  ]);
  expect(metadata.twitter?.images).toEqual([
    "http://blog.localhost:8000/first-post/card.png",
  ]);
});

test("an uploaded override is the preview instead", () => {
  const metadata = postMetadata({ ...POST, linkCardImage: OVERRIDE });
  expect(metadata.openGraph?.images).toEqual([
    {
      url: "http://blog.localhost:8000/media/5d31391b-cf7e-478f-814b-c6b28639e5a3/og/1",
      width: 1200,
      height: 630,
      alt: "Illarin keeps its own writing",
    },
  ]);
});

test("the article data a search engine reads invents no author or date", () => {
  const article = JSON.parse(postStructuredData(POST));
  expect(article["@type"]).toBe("Article");
  expect(article.headline).toBe("Illarin keeps its own writing");
  expect(article.url).toBe("http://blog.localhost:8000/first-post");
  expect(article.datePublished).toBe("2026-08-29T10:33:48Z");
  expect(article.dateModified).toBeUndefined();
  expect(article.author).toEqual({
    "@type": "Person",
    name: "Wren Ashdown",
    url: "http://localhost:8000/@wren",
  });
  expect(article.publisher).toEqual({
    "@type": "Organization",
    name: "Illarin",
    url: "http://localhost:8000",
  });
});

test("the article data carries the update it has and nothing more", () => {
  const article = JSON.parse(
    postStructuredData({ ...POST, updatedAt: "2026-09-02T11:15:00Z" }),
  );
  expect(article.dateModified).toBe("2026-09-02T11:15:00Z");
  expect(article.image).toBe("http://blog.localhost:8000/first-post/card.png");
});

test("a narrowed archive is canonical on the blog origin and offers its own feeds", () => {
  const metadata = blogMetadata(
    "Release",
    archiveDescription("Release"),
    "/category/release/page/2",
    "/category/release",
  );
  expect(metadata.alternates?.canonical).toBe(
    "http://blog.localhost:8000/category/release/page/2",
  );
  expect(metadata.alternates?.types).toEqual({
    "application/rss+xml":
      "http://blog.localhost:8000/category/release/feed.xml",
    "application/feed+json":
      "http://blog.localhost:8000/category/release/feed.json",
  });
});

test("a title cannot close the script the article data is written in", () => {
  const attacked = postStructuredData({
    ...POST,
    title: "</script><script>alert(1)</script>",
  });
  expect(attacked).not.toContain("<");
  expect(attacked).not.toContain(">");
  expect(attacked).not.toContain("&");
  expect(JSON.parse(attacked).headline).toBe(
    "</script><script>alert(1)</script>",
  );
});
