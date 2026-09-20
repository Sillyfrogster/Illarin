import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import type { BrowsePage } from "@/lib/api/query";
import { BrowseChapter } from "./BrowseChapter";

test("the landing gallery keeps public work links and respects adult cover preferences", () => {
  const page: BrowsePage = {
    items: [
      {
        id: "00000000-0000-4000-8000-000000000001",
        name: "Test character",
        creator: "example_creator",
        type: "character",
        isNsfw: true,
        cover: { url: "/test-cover.webp", width: 600, height: 900 },
      },
    ],
    facets: [],
    apps: [],
    total: 1,
    suppressed: 0,
    emptyState: null,
    nsfwPreference: "blurred",
  };
  const blurred = renderToStaticMarkup(<BrowseChapter page={page} />);
  expect(blurred).toContain(
    'href="/a/00000000-0000-4000-8000-000000000001/test-character"',
  );
  expect(blurred).toContain('href="/@example_creator"');
  expect(blurred).toContain('class="blur-xl"');
  expect(
    renderToStaticMarkup(
      <BrowseChapter page={{ ...page, nsfwPreference: "shown" }} />,
    ),
  ).not.toContain('class="blur-xl"');
  const hidden = renderToStaticMarkup(
    <BrowseChapter
      page={{
        ...page,
        items: [],
        suppressed: 1,
        emptyState: "suppressed",
        nsfwPreference: "hidden",
      }}
    />,
  );
  expect(hidden).not.toContain("test-cover.webp");
  expect(hidden).toContain('href="/browse"');
});
