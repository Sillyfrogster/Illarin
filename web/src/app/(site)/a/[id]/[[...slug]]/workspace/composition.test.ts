import { describe, expect, test } from "bun:test";
import type { AddableBlock, AssetBlock, AssetElement } from "@/lib/api/query";
import {
  arrangementRequest,
  blockDestinations,
  moveBlock,
  moveElement,
  offerGroups,
  relaidBlock,
  removeElement,
  seatElements,
} from "./composition";

function element(id: string, extra: Partial<AssetElement> = {}): AssetElement {
  return {
    content: { text: "" },
    facts: [],
    id,
    isEmpty: true,
    label: id,
    pinned: false,
    slot: "main",
    type: "prose",
    ...extra,
  } as AssetElement;
}

function block(id: string, extra: Partial<AssetBlock> = {}): AssetBlock {
  return {
    allowedLayouts: ["single", "duo", "stack-2", "trio", "stack-3"],
    definition: id,
    elements: [],
    hidden: false,
    hideable: true,
    id,
    isEmpty: true,
    layout: "single",
    position: 0,
    required: false,
    title: id,
    titleIsDefault: true,
    width: "full",
    ...extra,
  } as AssetBlock;
}

describe("moveBlock", () => {
  const page = [block("a"), block("b"), block("c")];

  test("puts a block where it was asked for and renumbers the page", () => {
    expect(moveBlock(page, "c", 0).map((item) => item.id)).toEqual([
      "c",
      "a",
      "b",
    ]);
    expect(moveBlock(page, "c", 0).map((item) => item.position)).toEqual([
      0, 1, 2,
    ]);
  });

  test("moves a block later", () => {
    expect(moveBlock(page, "a", 2).map((item) => item.id)).toEqual([
      "b",
      "c",
      "a",
    ]);
  });

  test("leaves the page alone where the move has nowhere to go", () => {
    expect(moveBlock(page, "a", 0)).toBe(page);
    expect(moveBlock(page, "a", -1)).toBe(page);
    expect(moveBlock(page, "a", 3)).toBe(page);
    expect(moveBlock(page, "nothing", 1)).toBe(page);
  });

  test("does not change what it was given", () => {
    moveBlock(page, "c", 0);
    expect(page.map((item) => item.id)).toEqual(["a", "b", "c"]);
  });
});

describe("blockDestinations", () => {
  const page = [block("a"), block("b"), block("c")];

  test("names every place a block can go by the block it lands beside", () => {
    expect(blockDestinations(page, "a")).toEqual([
      { label: "Before “c”", position: 1 },
      { label: "After “c”", position: 2 },
    ]);
  });

  test("leaves out the place the block already holds", () => {
    expect(blockDestinations(page, "b")).toEqual([
      { label: "Before “a”", position: 0 },
      { label: "After “c”", position: 2 },
    ]);
  });

  test("offers nothing where a block is the only one on the page", () => {
    expect(blockDestinations([block("a")], "a")).toEqual([]);
  });
});

describe("seatElements", () => {
  test("seats elements in the layout's own slots, in reading order", () => {
    const seated = seatElements("trio", [
      element("one", { slot: "middle" }),
      element("two", { slot: "left" }),
    ]);
    expect(seated.map((item) => item.slot)).toEqual(["left", "middle"]);
  });

  test("keeps an element that already sits where it belongs", () => {
    const elements = [element("one", { slot: "main" })];
    expect(seatElements("single", elements)[0]).toBe(elements[0]);
  });

  test("holds an element the layout has no slot for", () => {
    const seated = seatElements("single", [element("one"), element("two")]);
    expect(seated.map((item) => item.slot)).toEqual(["main", "main"]);
  });
});

describe("relaidBlock", () => {
  test("reseats the elements the new layout has to hold", () => {
    const relaid = relaidBlock(
      block("a", {
        elements: [
          element("one", { slot: "top" }),
          element("two", { slot: "bottom" }),
        ],
        layout: "stack-2",
      }),
      "duo",
    );
    expect(relaid.layout).toBe("duo");
    expect(relaid.elements.map((item) => item.slot)).toEqual(["left", "right"]);
  });
});

describe("moveElement", () => {
  const held = block("a", {
    elements: [element("one"), element("two"), element("three")],
    layout: "trio",
  });

  test("moves an element and reseats every one of them", () => {
    const moved = moveElement(held, "three", 0);
    expect(moved.elements.map((item) => item.id)).toEqual([
      "three",
      "one",
      "two",
    ]);
    expect(moved.elements.map((item) => item.slot)).toEqual([
      "left",
      "middle",
      "right",
    ]);
  });

  test("leaves the block alone where the move has nowhere to go", () => {
    expect(moveElement(held, "one", -1)).toBe(held);
    expect(moveElement(held, "three", 3)).toBe(held);
    expect(moveElement(held, "nothing", 1)).toBe(held);
  });

  test("moves a pinned element, because pinning holds it in its block", () => {
    const pinned = block("a", {
      elements: [element("one", { pinned: true }), element("two")],
      layout: "stack-2",
    });
    expect(
      moveElement(pinned, "one", 1).elements.map((item) => item.id),
    ).toEqual(["two", "one"]);
  });
});

describe("removeElement", () => {
  const held = block("a", {
    elements: [
      element("one"),
      element("two", { pinned: true }),
      element("three"),
    ],
    layout: "trio",
  });

  test("takes an element out and reseats the rest", () => {
    const left = removeElement(held, "one");
    expect(left.elements.map((item) => item.id)).toEqual(["two", "three"]);
    expect(left.elements.map((item) => item.slot)).toEqual(["left", "middle"]);
  });

  test("keeps a pinned element", () => {
    expect(removeElement(held, "two")).toBe(held);
  });

  test("keeps the block whole where the element is not in it", () => {
    expect(removeElement(held, "nothing")).toBe(held);
  });
});

describe("offerGroups", () => {
  const addable = [
    {
      choices: [{ label: "Text", type: "prose" }],
      definition: "gallery",
      group: "reader",
      groupTitle: "Things a reader sees",
      repeatable: false,
      summary: "Pictures a reader can open.",
      title: "Gallery",
    },
    {
      choices: [
        { label: "Text", type: "prose" },
        { label: "Links", type: "link_list" },
      ],
      definition: "custom_block",
      group: "other",
      groupTitle: "Anything else",
      repeatable: true,
      summary: "A heading you write.",
      title: "New block",
    },
  ] as AddableBlock[];

  test("groups the catalog by where its content ends up", () => {
    expect(offerGroups(addable, [], "").map((group) => group.title)).toEqual([
      "Things a reader sees",
      "Anything else",
    ]);
  });

  test("marks a block the page already holds", () => {
    const groups = offerGroups(
      addable,
      [block("g", { definition: "gallery" })],
      "",
    );
    expect(groups[0].offers[0].alreadyOn).toBe(true);
  });

  test("keeps offering a repeatable block the page already holds", () => {
    const groups = offerGroups(
      addable,
      [block("c", { definition: "custom_block" })],
      "",
    );
    expect(groups[1].offers[0].alreadyOn).toBe(false);
  });

  test("matches a search against the title and the summary", () => {
    expect(offerGroups(addable, [], "pictures")).toHaveLength(1);
    expect(offerGroups(addable, [], "GALLERY")[0].offers).toHaveLength(1);
    expect(offerGroups(addable, [], "nothing here")).toEqual([]);
  });
});

describe("arrangementRequest", () => {
  const saved = [block("a", { width: "half" }), block("b", { width: "third" })];

  test("takes each width from the saved copy, not the draft", () => {
    const draft = [block("a", { width: "full" }), block("b")];
    expect(arrangementRequest(draft, saved).blocks).toEqual([
      { hidden: false, id: "a", width: "half" },
      { hidden: false, id: "b", width: "third" },
    ]);
  });

  test("saves the order it was given", () => {
    const draft = [saved[1], saved[0]];
    expect(
      arrangementRequest(draft, saved).blocks.map((one) => one.id),
    ).toEqual(["b", "a"]);
  });

  test("carries the hidden state the draft holds", () => {
    const draft = [block("a", { hidden: true, width: "full" }), block("b")];
    expect(arrangementRequest(draft, saved).blocks[0].hidden).toBe(true);
  });

  test("keeps a block's own width where the saved copy has never seen it", () => {
    const added = [block("c", { width: "third" })];
    expect(arrangementRequest(added, saved).blocks).toEqual([
      { hidden: false, id: "c", width: "third" },
    ]);
  });
});
