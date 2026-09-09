import { describe, expect, test } from "bun:test";
import {
  chosenIndex,
  itemKeys,
  keyAfterMove,
  moveItem,
  readLines,
  replaceAt,
  without,
  writeLines,
} from "./collection";

describe("itemKeys", () => {
  test("names a saved item by its own id", () => {
    expect(itemKeys([{ id: "a1" }, { id: "b2" }])).toEqual(["a1", "b2"]);
  });

  test("names an unsaved item by the position it was added at", () => {
    expect(itemKeys([{ id: "a1" }, {}, {}])).toEqual(["a1", "new:1", "new:2"]);
  });
});

describe("chosenIndex", () => {
  const keys = ["a1", "b2", "c3"];

  test("finds the chosen item wherever it has moved to", () => {
    expect(chosenIndex(["c3", "a1", "b2"], "b2")).toBe(2);
  });

  test("holds the first item when nothing is chosen", () => {
    expect(chosenIndex(keys, null)).toBe(0);
  });

  test("resolves an unsaved choice onto the item the save gave an id to", () => {
    expect(chosenIndex(["a1", "b2", "c3"], "new:2")).toBe(2);
  });

  test("falls back to the position a removed item held", () => {
    expect(chosenIndex(["a1", "c3"], "b2")).toBe(0);
  });

  test("answers -1 for an empty collection", () => {
    expect(chosenIndex([], "a1")).toBe(-1);
  });

  test("clamps an unsaved choice past the end of the collection", () => {
    expect(chosenIndex(["a1"], "new:4")).toBe(0);
  });
});

describe("keyAfterMove", () => {
  test("a saved item keeps its own id wherever it lands", () => {
    expect(keyAfterMove("a1", 3)).toBe("a1");
  });

  test("an unsaved item is known by where it landed", () => {
    expect(keyAfterMove("new:2", 1)).toBe("new:1");
  });
});

describe("moveItem", () => {
  test("moves an item later without disturbing the rest", () => {
    expect(moveItem(["a", "b", "c"], 0, 1)).toEqual(["b", "a", "c"]);
  });

  test("moves an item earlier", () => {
    expect(moveItem(["a", "b", "c"], 2, 1)).toEqual(["a", "c", "b"]);
  });

  test("leaves the collection alone when the move has nowhere to go", () => {
    expect(moveItem(["a", "b"], 0, -1)).toEqual(["a", "b"]);
    expect(moveItem(["a", "b"], 1, 2)).toEqual(["a", "b"]);
  });

  test("does not mutate what it was given", () => {
    const items = ["a", "b"];
    moveItem(items, 0, 1);
    expect(items).toEqual(["a", "b"]);
  });
});

describe("replaceAt and without", () => {
  test("replaceAt changes one item and copies the collection", () => {
    const items = [{ text: "one" }, { text: "two" }];
    expect(replaceAt(items, 1, { text: "changed" })).toEqual([
      { text: "one" },
      { text: "changed" },
    ]);
    expect(items[1].text).toBe("two");
  });

  test("without drops one item", () => {
    expect(without(["a", "b", "c"], 1)).toEqual(["a", "c"]);
  });
});

describe("readLines and writeLines", () => {
  test("reading drops blank lines and surrounding space", () => {
    expect(readLines(" one \n\n  two\n")).toEqual(["one", "two"]);
  });

  test("writing puts one item on each line", () => {
    expect(writeLines(["one", "two"])).toBe("one\ntwo");
  });

  test("writing nothing gives an empty field", () => {
    expect(writeLines(undefined)).toBe("");
  });
});
