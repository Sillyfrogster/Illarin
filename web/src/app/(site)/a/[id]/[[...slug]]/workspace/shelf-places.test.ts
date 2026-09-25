import { describe, expect, test } from "bun:test";
import type { ShelfImport, ShelfPiece, WorkBlock } from "@/lib/api/query";
import {
  importCounts,
  importLabel,
  placeChoices,
  positionAt,
  withGhost,
} from "./shelf-places";

function prose(id: string, extra: Record<string, unknown> = {}) {
  return {
    id,
    type: "prose",
    label: "",
    role: null,
    fromFile: false,
    ...extra,
  };
}

function block(id: string, title: string, elements: unknown[]): WorkBlock {
  return { id, title, elements } as unknown as WorkBlock;
}

function piece(kind: "section" | "picture"): ShelfPiece {
  return { id: kind, kind, section: "", heading: "" };
}

describe("placeChoices", () => {
  const page = [
    block("a", "About", [prose("a1")]),
    block("b", "Looks", [{ ...prose("b1"), type: "image_set" }]),
    block("c", "Lore", [
      prose("c1", { label: "Description", role: "description" }),
      prose("c2", { fromFile: true }),
    ]),
  ];

  test("offers a new block first and after every block", () => {
    expect(placeChoices(page).blocks).toEqual([
      { position: 0, label: "First on the page" },
      { position: 1, label: "After “About”" },
      { position: 2, label: "After “Looks”" },
      { position: 3, label: "After “Lore”" },
    ]);
  });

  test("offers only text elements the creator can write in", () => {
    expect(placeChoices(page).texts).toEqual([
      { elementId: "a1", blockId: "a", label: "“About”" },
      { elementId: "c1", blockId: "c", label: "Description in “Lore”" },
    ]);
  });
});

describe("importLabel", () => {
  const held = (source: "pasted" | "readme", title: string) =>
    ({ source, title, createdAt: "2026-09-24T10:00:00Z" }) as ShelfImport;

  test("names a paste by its day and its title", () => {
    expect(importLabel(held("pasted", "Aria"))).toBe("Pasted 24 Sep · Aria");
    expect(importLabel(held("pasted", " "))).toBe("Pasted 24 Sep");
  });

  test("names a README import by where it came from", () => {
    expect(importLabel(held("readme", "Tool"))).toBe("From README");
  });
});

test("importCounts names each kind it holds", () => {
  expect(
    importCounts([piece("section"), piece("section"), piece("picture")]),
  ).toBe("2 sections · 1 picture");
  expect(importCounts([piece("picture"), piece("picture")])).toBe("2 pictures");
});

describe("positionAt", () => {
  const rect = { top: 100, height: 200 };
  test("lands before a block over its upper half and after it over the lower", () => {
    expect(positionAt(3, 150, rect)).toBe(3);
    expect(positionAt(3, 250, rect)).toBe(4);
  });
});

test("withGhost opens the ghost at the position without moving the rest", () => {
  expect(withGhost(["a", "b"], 1, "ghost")).toEqual(["a", "ghost", "b"]);
  expect(withGhost(["a", "b"], 2, "ghost")).toEqual(["a", "b", "ghost"]);
  expect(withGhost(["a", "b"], null, "ghost")).toEqual(["a", "b"]);
});
