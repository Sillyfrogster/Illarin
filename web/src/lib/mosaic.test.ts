import { expect, test } from "bun:test";
import {
  MOSAIC_GAP_PX,
  type MosaicPicture,
  mosaicRows,
  pictureWidth,
} from "./mosaic";

function picture(id: string, width: number, height: number): MosaicPicture {
  return { height, id, src: `/${id}`, width };
}

function rowWidth(pictures: MosaicPicture[], height: number): number {
  const widths = pictures.reduce(
    (total, item) => total + pictureWidth(item, height),
    0,
  );
  return widths + MOSAIC_GAP_PX * (pictures.length - 1);
}

test("fills a full row to the width available", () => {
  const [row] = mosaicRows(
    [picture("a", 100, 100), picture("b", 100, 100), picture("c", 100, 100)],
    { rowHeight: 200, width: 600 },
  );

  expect(row.justified).toBe(true);
  expect(rowWidth(row.pictures, row.height)).toBeCloseTo(600, 5);
});

test("keeps every picture, in order", () => {
  const rows = mosaicRows(
    [
      picture("a", 900, 600),
      picture("b", 600, 900),
      picture("c", 900, 600),
      picture("d", 600, 900),
      picture("e", 900, 600),
    ],
    { rowHeight: 180, width: 620 },
  );

  expect(rows.flatMap((row) => row.pictures).map((item) => item.id)).toEqual([
    "a",
    "b",
    "c",
    "d",
    "e",
  ]);
});

test("leaves one leftover picture at the height the creator chose", () => {
  const rows = mosaicRows(
    [
      picture("a", 900, 600),
      picture("b", 900, 600),
      picture("c", 900, 600),
      picture("d", 900, 600),
    ],
    { rowHeight: 190, width: 620 },
  );

  const last = rows.at(-1);
  expect(last?.pictures.map((item) => item.id)).toEqual(["d"]);
  expect(last?.justified).toBe(false);
  expect(last?.height).toBe(190);
});

test("stretches a short row that lands near the chosen height", () => {
  const rows = mosaicRows(
    [picture("a", 300, 200), picture("b", 300, 200), picture("c", 300, 200)],
    { rowHeight: 150, width: 620 },
  );

  const last = rows.at(-1);
  expect(last?.justified).toBe(true);
  expect(last?.height).toBeLessThanOrEqual(150 * 1.35);
});

test("treats a picture with no recorded size as square", () => {
  const rows = mosaicRows([picture("a", 0, 0), picture("b", 0, 0)], {
    rowHeight: 100,
    width: 208,
  });

  expect(rows).toHaveLength(1);
  expect(rows[0].height).toBeCloseTo(100, 5);
});

test("waits for a measured width before packing", () => {
  const rows = mosaicRows([picture("a", 100, 100), picture("b", 100, 100)], {
    rowHeight: 200,
    width: 0,
  });

  expect(rows).toHaveLength(1);
  expect(rows[0].justified).toBe(false);
  expect(rows[0].pictures).toHaveLength(2);
});
