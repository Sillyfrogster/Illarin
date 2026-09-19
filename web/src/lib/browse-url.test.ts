import { describe, expect, test } from "bun:test";
import { buildBrowseHref, readBrowseFilters } from "./browse-url";

describe("browse URL state", () => {
  test("round-trips the complete search expression through one q parameter", () => {
    const expression = '  moon tag:"original character" mood:gentle  ';
    const href = buildBrowseHref({
      type: "character",
      app: "raw",
      q: expression,
      facet: ["tone=gentle"],
    });
    const url = new URL(href, "https://illarin.test");

    expect(url.searchParams.getAll("q")).toEqual([expression]);
    expect(readBrowseFilters(Object.fromEntries(url.searchParams))).toEqual({
      type: "character",
      app: "raw",
      q: expression,
      facet: ["tone=gentle"],
    });
  });

  test("still reads a link that named the app under its old name", () => {
    expect(readBrowseFilters({ platform: "sillytavern" }).app).toBe(
      "sillytavern",
    );
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
});
