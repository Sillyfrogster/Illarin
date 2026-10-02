import { describe, expect, test } from "bun:test";
import { dropPosition, snapWidth } from "./gestures";

const GRID = { left: 0, width: 1240, gap: 20 };
// Each track is (1240 - 11 * 20) / 12 = 85px, so a column step is 105px.

describe("snapWidth", () => {
  test("snaps the dragged edge to the nearest width the block may take", () => {
    expect(
      snapWidth({ ...GRID, edge: 620, startColumn: 1, layout: "single" }),
    ).toBe("half");
    expect(
      snapWidth({ ...GRID, edge: 1230, startColumn: 1, layout: "single" }),
    ).toBe("full");
    expect(
      snapWidth({ ...GRID, edge: 90, startColumn: 1, layout: "single" }),
    ).toBe("third");
  });

  test("measures from the column the block starts in", () => {
    expect(
      snapWidth({ ...GRID, edge: 1240, startColumn: 9, layout: "single" }),
    ).toBe("third");
  });

  test("never offers a width the block's layout cannot fit", () => {
    expect(
      snapWidth({ ...GRID, edge: 300, startColumn: 1, layout: "duo" }),
    ).toBe("two_thirds");
    expect(
      snapWidth({ ...GRID, edge: 300, startColumn: 1, layout: "trio" }),
    ).toBe("full");
  });
});

const box = (left: number, top: number, width: number, height: number) => ({
  bottom: top + height,
  left,
  right: left + width,
  top,
});

describe("dropPosition", () => {
  // Two rows: a and b side by side, then c alone.
  const rest = [
    box(0, 0, 600, 400),
    box(620, 0, 600, 200),
    box(0, 500, 1240, 300),
  ];

  test("lands before a block when the pointer is on its first half in the same row", () => {
    expect(dropPosition(rest, { x: 100, y: 50 })).toBe(0);
    expect(dropPosition(rest, { x: 700, y: 50 })).toBe(1);
  });

  test("counts a shorter block in the row as passed once the pointer is right of its middle", () => {
    expect(dropPosition(rest, { x: 1000, y: 350 })).toBe(2);
  });

  test("lands after everything below the last row", () => {
    expect(dropPosition(rest, { x: 100, y: 1200 })).toBe(3);
    expect(dropPosition(rest, { x: 800, y: 600 })).toBe(3);
  });

  test("counts the pointer as in a row from just above its top, where a block's handle sits", () => {
    expect(dropPosition(rest, { x: 900, y: 480 }, 40)).toBe(3);
    expect(dropPosition(rest, { x: 100, y: 480 }, 40)).toBe(2);
  });

  test("lands first above the page", () => {
    expect(dropPosition(rest, { x: 900, y: -100 })).toBe(0);
  });
});
