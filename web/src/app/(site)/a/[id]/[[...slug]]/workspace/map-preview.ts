import type { WorkElement } from "@/lib/api/query";

/** How many items of a list a block shows on the arrange map; the tile is cut off below them anyway. */
const ITEMS = 4;
/** How much of a long text a block shows on the arrange map. */
const CHARACTERS = 1200;

const previews = new WeakMap<WorkElement, WorkElement>();

function trim(value: unknown, depth: number): unknown {
  if (typeof value === "string")
    return value.length > CHARACTERS ? value.slice(0, CHARACTERS) : value;
  if (Array.isArray(value))
    return value.slice(0, ITEMS).map((item) => trim(item, depth - 1));
  if (value && typeof value === "object" && depth > 0)
    return Object.fromEntries(
      Object.entries(value).map(([key, item]) => [key, trim(item, depth - 1)]),
    );
  return value;
}

/** previewElement is an element cut down to what fits on its arrange map tile, so the map draws a few items instead of the whole block. */
export function previewElement(element: WorkElement): WorkElement {
  const cached = previews.get(element);
  if (cached) return cached;
  const preview = {
    ...element,
    content: trim(element.content, 3),
  } as WorkElement;
  previews.set(element, preview);
  return preview;
}
