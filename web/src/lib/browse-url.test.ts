import { describe, expect, test } from "bun:test";
import { buildBrowseHref, chooseFilter, readBrowseFilters } from "./browse-url";

describe("browse URL state", () => {
  test("round-trips the complete search expression through one q parameter", () => {
    const expression = '  moon tag:"original character" mood:gentle  ';
    const href = buildBrowseHref({
      type: "character",
      q: expression,
      facet: ["tone=gentle"],
    });
    const url = new URL(href, "https://illarin.test");

    expect(url.searchParams.getAll("q")).toEqual([expression]);
    expect(readBrowseFilters(Object.fromEntries(url.searchParams))).toEqual({
      type: "character",
      q: expression,
      facet: ["tone=gentle"],
    });
  });

  test("keeps creator profile filters on the creator profile", () => {
    expect(
      buildBrowseHref({ type: "lorebook", q: "moonlit" }, "/@verified.creator"),
    ).toBe("/@verified.creator?type=lorebook&q=moonlit");
  });

  test("keeps Pack as a type", () => {
    expect(readBrowseFilters({ type: "pack" }).type).toBe("pack");
    expect(buildBrowseHref({ type: "pack" })).toBe("/browse?type=pack");
  });

  test("still reads the type from links that say kind", () => {
    expect(readBrowseFilters({ kind: "lorebook" }).type).toBe("lorebook");
  });

  test("choosing a filter replaces its earlier value and keeps the others", () => {
    const filters = {
      type: "character" as const,
      q: "moon",
      facet: ["lorebook=true", "alternate_greetings=1"],
    };

    expect(chooseFilter(filters, "alternate_greetings", "2-4")).toEqual({
      ...filters,
      facet: ["lorebook=true", "alternate_greetings=2-4"],
    });
    expect(chooseFilter(filters, "lorebook", null).facet).toEqual([
      "alternate_greetings=1",
    ]);
    expect(
      chooseFilter({ facet: ["lorebook=true"] }, "lorebook", null).facet,
    ).toBeUndefined();
  });
});
