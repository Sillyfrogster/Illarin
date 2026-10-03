import { expect, test } from "bun:test";
import type { BrowsePage, BrowseType, BrowseWork } from "@/lib/api/query";
import { HOME_TYPES, homeRows } from "./rows";

function page(count: number, extra: Partial<BrowsePage> = {}): BrowsePage {
  const items = Array.from(
    { length: count },
    (_, at): BrowseWork => ({
      id: `00000000-0000-4000-8000-${String(at).padStart(12, "0")}`,
      name: `Work ${at}`,
      apps: [],
      creator: "example_creator",
      type: "character",
      isNsfw: false,
      cover: null,
      viewCount: 0,
      downloadCount: 0,
      tags: [],
    }),
  );
  return {
    items,
    facets: [],
    app: null,
    apps: [],
    types: [],
    allTypes: count,
    total: count,
    suppressed: 0,
    emptyState: count ? null : "nothing_published",
    nsfwPreference: "blurred",
    ...extra,
  };
}

function lists(pages: Partial<Record<BrowseType, BrowsePage | null>>) {
  return HOME_TYPES.map((type) => ({ type, page: pages[type] ?? page(0) }));
}

test("the home page never lists packs", () => {
  expect(HOME_TYPES).not.toContain("pack");
});

test("a type with fewer than six works has no row", () => {
  const home = homeRows(
    lists({ character: page(12), lorebook: page(5), preset: page(6) }),
  );
  expect(home.state).toBe("rows");
  expect(home.rows.map((row) => row.type)).toEqual(["character", "preset"]);
});

test("rows hidden by the adult setting say so instead of looking empty", () => {
  const hidden = homeRows(
    lists({ character: page(0, { suppressed: 9, emptyState: "suppressed" }) }),
  );
  expect(hidden.state).toBe("hidden");

  const partly = homeRows(lists({ character: page(8, { suppressed: 2 }) }));
  expect(partly.state).toBe("rows");
  expect(partly.someHidden).toBe(true);
});

test("the page tells apart nothing published, too few for a row, and a failed load", () => {
  expect(homeRows(lists({})).state).toBe("empty");
  expect(homeRows(lists({ character: page(3) })).state).toBe("few");
  expect(homeRows(HOME_TYPES.map((type) => ({ type, page: null }))).state).toBe(
    "failed",
  );
});
