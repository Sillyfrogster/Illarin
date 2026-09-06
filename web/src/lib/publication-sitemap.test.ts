import { expect, test } from "bun:test";
import type { PostSummary } from "@/lib/api/query";
import { publicationSitemap } from "@/lib/publication-sitemap";

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

const WHOLE = { address: "/blog", pages: 2 };

function addressesIn(entries: { url: string }[]): string[] {
  return entries.map((entry) => entry.url);
}

test("the publication and every numbered page of it are crawlable", () => {
  const entries = addressesIn(publicationSitemap([POST], [WHOLE]));
  expect(entries).toContain("http://localhost:8000/blog");
  expect(entries).toContain("http://localhost:8000/blog/page/2");
  expect(entries).not.toContain("http://localhost:8000/blog/page/1");
});

test("a narrowed archive is crawlable at its own address", () => {
  const entries = addressesIn(
    publicationSitemap(
      [POST],
      [
        WHOLE,
        { address: "/blog/category/article", pages: 1 },
        { address: "/blog/app/lumiverse", pages: 3 },
      ],
    ),
  );
  expect(entries).toContain("http://localhost:8000/blog/category/article");
  expect(entries).toContain("http://localhost:8000/blog/app/lumiverse/page/3");
});

test("a post is crawlable at the address it answers to now", () => {
  const entries = addressesIn(
    publicationSitemap([{ ...POST, slug: "corrected-address" }], [WHOLE]),
  );
  expect(entries).toContain("http://localhost:8000/blog/corrected-address");
  expect(entries).not.toContain("http://localhost:8000/blog/first-post");
});

test("a post carries the day of its last public change", () => {
  const entry = publicationSitemap([POST], [WHOLE]).at(-1);
  expect(entry?.lastModified).toEqual(new Date("2026-08-29T10:33:48Z"));

  const revised = publicationSitemap(
    [{ ...POST, updatedAt: "2026-09-02T11:15:00Z" }],
    [WHOLE],
  ).at(-1);
  expect(revised?.lastModified).toEqual(new Date("2026-09-02T11:15:00Z"));
});

test("an empty publication is still crawlable and lists no post", () => {
  const entries = publicationSitemap([], [{ address: "/blog", pages: 0 }]);
  expect(addressesIn(entries)).toEqual(["http://localhost:8000/blog"]);
});

test("nothing a reader cannot reach is listed", () => {
  const listed = addressesIn(
    publicationSitemap(
      [POST],
      [WHOLE, { address: "/blog/category/article", pages: 1 }],
    ),
  ).join(" ");
  for (const private_ of ["/admin", "feed.", "sitemap", "/preview", "/edit"]) {
    expect(listed).not.toContain(private_);
  }
});
