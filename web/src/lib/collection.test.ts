import { describe, expect, test } from "bun:test";
import {
  BROWSE_THRESHOLD,
  BROWSE_WEIGHT,
  browsable,
  type CollectionItem,
  type CollectionView,
  viewCollection,
} from "./collection";

function item(
  key: string,
  name: string,
  more: Partial<CollectionItem> = {},
): CollectionItem {
  return { detail: null, key, name, ...more };
}

const showing: CollectionView = {
  includeOff: true,
  order: "given",
  search: "",
};

describe("when a run earns a browser", () => {
  test("a short run stays on the page", () => {
    const short = Array.from({ length: BROWSE_THRESHOLD }, (_, n) =>
      item(`${n}`, `Item ${n}`),
    );
    expect(browsable(short)).toBe(false);
  });

  test("a long run of named things opens in a browser", () => {
    const long = Array.from({ length: BROWSE_THRESHOLD + 1 }, (_, n) =>
      item(`${n}`, `Item ${n}`),
    );
    expect(browsable(long)).toBe(true);
  });

  test("a long run that cannot name its items is a sequence, not a collection", () => {
    const turns = Array.from({ length: 40 }, (_, n) => item(`${n}`, ""));
    expect(browsable(turns)).toBe(false);
  });

  test("a few named passages that outweigh a page open in a browser", () => {
    const passages = Array.from({ length: 4 }, (_, n) =>
      item(`${n}`, `Part ${n}`, { weight: BROWSE_WEIGHT / 3 }),
    );
    expect(browsable(passages)).toBe(true);
  });

  test("a few short named passages stay on the page", () => {
    const passages = Array.from({ length: 4 }, (_, n) =>
      item(`${n}`, `Part ${n}`, { weight: 40 }),
    );
    expect(browsable(passages)).toBe(false);
  });

  test("heavy passages that cannot all be named stay on the page", () => {
    const greetings = Array.from({ length: 4 }, (_, n) =>
      item(`${n}`, "", { weight: BROWSE_WEIGHT }),
    );
    expect(browsable(greetings)).toBe(false);
  });

  test("one unnamed item is enough to keep a run on the page", () => {
    const mixed = Array.from({ length: 40 }, (_, n) =>
      item(`${n}`, n === 7 ? "  " : `Item ${n}`),
    );
    expect(browsable(mixed)).toBe(false);
  });
});

describe("the index a reader searches", () => {
  const book = [
    item("a", "Remielle Dan", { terms: ["ramiel", "void hunter"] }),
    item("b", "Belle", { off: true, terms: ["belle"] }),
    item("c", "Aokigahara", { group: "Places", terms: ["forest"] }),
  ];

  test("a search matches a name", () => {
    expect(
      viewCollection(book, { ...showing, search: "remielle" }).map(
        (entry) => entry.key,
      ),
    ).toEqual(["a"]);
  });

  test("a search matches a term, which is how a reader finds an entry it never names", () => {
    expect(
      viewCollection(book, { ...showing, search: "FOREST" }).map(
        (entry) => entry.key,
      ),
    ).toEqual(["c"]);
  });

  test("a search matches a group", () => {
    expect(
      viewCollection(book, { ...showing, search: "places" }).map(
        (entry) => entry.key,
      ),
    ).toEqual(["c"]);
  });

  test("a search matching nothing shows nothing", () => {
    expect(viewCollection(book, { ...showing, search: "kraken" })).toEqual([]);
  });

  test("items that are off can be folded away", () => {
    expect(
      viewCollection(book, { ...showing, includeOff: false }).map(
        (entry) => entry.key,
      ),
    ).toEqual(["a", "c"]);
  });

  test("given order is the order the creator wrote", () => {
    expect(viewCollection(book, showing).map((entry) => entry.key)).toEqual([
      "a",
      "b",
      "c",
    ]);
  });

  test("ordering by name keeps ties in the order they were given", () => {
    const twins = [
      item("1", "Belle"),
      item("2", "Aokigahara"),
      item("3", "Belle"),
    ];
    expect(
      viewCollection(twins, { ...showing, order: "name" }).map(
        (entry) => entry.key,
      ),
    ).toEqual(["2", "1", "3"]);
  });
});
