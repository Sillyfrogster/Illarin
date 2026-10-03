import {
  BLOCK_WIDTHS,
  type BlockLayout,
  type BlockWidth,
  LAYOUTS,
  WIDTH_COLUMNS,
} from "@/lib/page-arrangement";

const GRID_COLUMNS = 12;

type Edges = { top: number; bottom: number; left: number; right: number };

/** snapWidth is the width a block takes when its right edge is dragged to `edge`, limited to the widths its layout fits. */
export function snapWidth({
  edge,
  gap,
  layout,
  left,
  startColumn,
  width,
}: {
  edge: number;
  gap: number;
  layout: BlockLayout;
  left: number;
  startColumn: number;
  width: number;
}): BlockWidth {
  const step = (width - gap * (GRID_COLUMNS - 1)) / GRID_COLUMNS + gap;
  const origin = left + (startColumn - 1) * step;
  const columns = (edge - origin + gap) / step;
  const allowed = BLOCK_WIDTHS.filter(
    (choice) => WIDTH_COLUMNS[choice] >= LAYOUTS[layout].minimumColumns,
  );
  return allowed.reduce((best, choice) =>
    Math.abs(WIDTH_COLUMNS[choice] - columns) <
    Math.abs(WIDTH_COLUMNS[best] - columns)
      ? choice
      : best,
  );
}

/** dropPosition is where a dragged block lands among the others, read in page order: earlier rows, then the pointer's side of each block in its own row, or of the middle of a block alone in its row. */
export function dropPosition(
  rest: Edges[],
  point: { x: number; y: number },
): number {
  const tops = [...new Set(rest.map((one) => one.top))].sort((a, b) => a - b);
  const rowTop = tops.filter((top) => top <= point.y).at(-1);
  if (rowTop === undefined) return 0;
  const row = rest.filter((one) => one.top === rowTop);
  const rowBottom = Math.max(...row.map((one) => one.bottom));
  const earlier = rest.filter((one) => one.top < rowTop).length;
  if (point.y > rowBottom) return earlier + row.length;
  if (row.length === 1)
    return earlier + (point.y >= (row[0].top + row[0].bottom) / 2 ? 1 : 0);
  return (
    earlier + row.filter((one) => point.x >= (one.left + one.right) / 2).length
  );
}
