import { describe, expect, test } from "bun:test";
import type { AssetBlock, AssetElement } from "@/lib/api/query";
import { destinationsIn } from "./JumpPalette";

function element(id: string, label: string): AssetElement {
  return {
    content: { text: "" },
    facts: [],
    id,
    isEmpty: false,
    label,
    pinned: false,
    slot: id,
    type: "prose",
  };
}

const blocks: AssetBlock[] = [
  {
    allowedLayouts: ["single"],
    definition: "character",
    elements: [element("a", "Description"), element("b", "Personality")],
    hidden: false,
    hideable: true,
    id: "one",
    isEmpty: false,
    layout: "single",
    position: 0,
    required: false,
    title: "The character",
    titleIsDefault: true,
    width: "full",
  },
];

describe("destinationsIn", () => {
  test("offers every block and every element it holds", () => {
    expect(destinationsIn(blocks)).toEqual([
      { blockId: "one", id: "one", label: "The character", where: "Block" },
      {
        blockId: "one",
        elementId: "a",
        id: "a",
        label: "Description",
        where: "The character",
      },
      {
        blockId: "one",
        elementId: "b",
        id: "b",
        label: "Personality",
        where: "The character",
      },
    ]);
  });

  test("offers nothing for a page with no blocks", () => {
    expect(destinationsIn([])).toEqual([]);
  });
});
