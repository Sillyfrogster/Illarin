import { describe, expect, test } from "bun:test";
import type { WorkBlock, WorkElement } from "@/lib/api/query";
import {
  blockSaveRequest,
  changedBlockIds,
  firstCursor,
  isEmptyContent,
  replaceBlock,
  replaceElement,
  writesInPlace,
} from "./save";

function element(
  id: string,
  type: WorkElement["type"],
  content: WorkElement["content"],
): WorkElement {
  return {
    content,
    facts: [],
    id,
    isEmpty: false,
    label: id,
    pinned: false,
    fromFile: false,
    slot: id,
    type,
  };
}

function block(id: string, elements: WorkElement[]): WorkBlock {
  return {
    allowedLayouts: ["single"],
    definition: "character",
    elements,
    hidden: false,
    hideable: true,
    id,
    isEmpty: false,
    layout: "single",
    position: 0,
    required: false,
    title: id,
    titleIsDefault: false,
    width: "full",
  };
}

describe("writesInPlace", () => {
  test("takes the element types the page itself can write", () => {
    expect(writesInPlace(element("a", "prose", { text: "" }))).toBe(true);
    expect(writesInPlace(element("b", "text_set", { texts: [] }))).toBe(true);
    expect(writesInPlace(element("c", "entry_table", { entries: [] }))).toBe(
      false,
    );
  });
});

describe("changedBlockIds", () => {
  test("names a block whose content was written", () => {
    const before = [block("one", [element("a", "prose", { text: "Old" })])];
    const after = [block("one", [element("a", "prose", { text: "New" })])];

    expect(changedBlockIds(after, before)).toEqual(["one"]);
  });

  test("names a block whose width or heading changed", () => {
    const before = [block("one", [])];

    expect(changedBlockIds([{ ...before[0], width: "half" }], before)).toEqual([
      "one",
    ]);
    expect(
      changedBlockIds([{ ...before[0], title: "Another" }], before),
    ).toEqual(["one"]);
  });

  test("ignores a block the server changed on its own", () => {
    const before = [block("one", [element("a", "prose", { text: "Same" })])];
    const after = [
      { ...before[0], facts: undefined, isEmpty: true } as WorkBlock,
    ];

    expect(changedBlockIds(after, before)).toEqual([]);
  });

  test("names a block that has no earlier copy", () => {
    expect(changedBlockIds([block("new", [])], [])).toEqual(["new"]);
  });
});

describe("blockSaveRequest", () => {
  test("sends null for a heading the creator left at its default", () => {
    const subject = { ...block("one", []), titleIsDefault: true };

    expect(blockSaveRequest(subject).title).toBeNull();
  });

  test("sends the creator's own heading", () => {
    expect(blockSaveRequest(block("one", [])).title).toBe("one");
  });

  test("carries the elements it is given rather than the block's own", () => {
    const subject = block("one", [element("a", "prose", { text: "Old" })]);
    const written = [element("a", "prose", { text: "New" })];

    expect(blockSaveRequest(subject, { elements: written })).toMatchObject({
      elements: [{ content: { text: "New" }, id: "a" }],
    });
  });
});

describe("isEmptyContent", () => {
  test("reads emptiness from the element's own body", () => {
    expect(isEmptyContent(element("a", "prose", { text: "   " }))).toBe(true);
    expect(isEmptyContent(element("a", "prose", { text: "Here" }))).toBe(false);
    expect(
      isEmptyContent(element("b", "text_set", { texts: [{ text: "" }] })),
    ).toBe(true);
    expect(
      isEmptyContent(element("c", "field_list", { fields: [{ value: "1" }] })),
    ).toBe(false);
  });
});

describe("firstCursor", () => {
  test("points at the first field of an element written in place", () => {
    expect(firstCursor(element("a", "prose", { text: "" }))).toBe("a:text");
    expect(firstCursor(element("b", "text_set", { texts: [] }))).toBe(
      "b:0:text",
    );
  });

  test("points nowhere for an element edited beside the page", () => {
    expect(
      firstCursor(element("c", "entry_table", { entries: [] })),
    ).toBeNull();
  });
});

describe("replaceBlock and replaceElement", () => {
  test("put the new copy in place of the old one", () => {
    const blocks = [block("one", []), block("two", [])];
    const saved = { ...blocks[1], title: "Renamed" };

    expect(replaceBlock(blocks, saved)[1].title).toBe("Renamed");
  });

  test("leave the other elements alone", () => {
    const subject = block("one", [
      element("a", "prose", { text: "A" }),
      element("b", "prose", { text: "B" }),
    ]);
    const written = element("b", "prose", { text: "Written" });

    expect(replaceElement(subject, written).elements).toMatchObject([
      { content: { text: "A" } },
      { content: { text: "Written" } },
    ]);
  });
});
