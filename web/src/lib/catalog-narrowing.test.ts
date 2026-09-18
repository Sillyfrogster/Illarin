import { describe, expect, test } from "bun:test";
import type { BrowsePage } from "./api/query";
import { narrowingsInForce } from "./catalog-narrowing";

const overview = {
  platforms: [
    { value: "sillytavern", label: "SillyTavern", count: 4, selected: false },
    { value: "risu", label: "RisuAI", count: 2, selected: false },
  ],
  facets: [
    {
      key: "tone",
      label: "Tone",
      options: [
        { value: "gentle", label: "Gentle", count: 3, selected: true },
        { value: "grim", label: "Grim", count: 1, selected: false },
      ],
    },
  ],
} as unknown as BrowsePage;

describe("what the catalog is narrowed by", () => {
  test("says nothing is in force when nothing is chosen", () => {
    expect(narrowingsInForce({}, overview)).toEqual([]);
  });

  test("names the kind and lifts it without keeping kind-scoped facets", () => {
    const [kind] = narrowingsInForce(
      { type: "lorebook", facet: ["tone=gentle"] },
      overview,
    );

    expect(kind.group).toBe("Kind");
    expect(kind.label).toBe("Lorebooks");
    expect(kind.without).toEqual({});
  });

  test("names an app by the label the catalog gave it", () => {
    const [app] = narrowingsInForce({ platform: "risu" }, overview);

    expect(app.label).toBe("RisuAI");
    expect(app.without).toEqual({});
  });

  test("falls back to the raw value when the catalog does not describe it", () => {
    const [app] = narrowingsInForce({ platform: "an-app-not-listed" }, null);

    expect(app.label).toBe("an-app-not-listed");
  });

  test("lifts one facet and leaves the others alone", () => {
    const facets = narrowingsInForce(
      { facet: ["tone=gentle", "tone=grim"] },
      overview,
    );

    expect(facets.map((one) => one.label)).toEqual(["Gentle", "Grim"]);
    expect(facets[0].without).toEqual({ facet: ["tone=grim"] });
    expect(facets[1].without).toEqual({ facet: ["tone=gentle"] });
  });

  test("carries a search as its own narrowing", () => {
    const [search] = narrowingsInForce({ q: "tag:fantasy" }, overview);

    expect(search.group).toBe("Search");
    expect(search.label).toBe("tag:fantasy");
    expect(search.without).toEqual({});
  });

  test("gives every narrowing a key of its own when two read alike", () => {
    const all = narrowingsInForce(
      {
        type: "theme",
        platform: "gentle",
        facet: ["tone=gentle"],
        q: "gentle",
      },
      overview,
    );

    expect(new Set(all.map((one) => one.id)).size).toBe(all.length);
  });
});
