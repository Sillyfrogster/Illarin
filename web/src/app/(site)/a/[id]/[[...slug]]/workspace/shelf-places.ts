import type { ShelfImport, ShelfPiece, WorkBlock } from "@/lib/api/query";
import { elementLabel } from "@/lib/element-label";

export type BlockChoice = { position: number; label: string };
export type TextChoice = { elementId: string; blockId: string; label: string };

/** placeChoices lists where a section can go: a new block at each position, or the end of each text element the creator writes in. */
export function placeChoices(blocks: WorkBlock[]): {
  blocks: BlockChoice[];
  texts: TextChoice[];
} {
  return {
    blocks: [
      { position: 0, label: "First on the page" },
      ...blocks.map((block, index) => ({
        position: index + 1,
        label: `After “${block.title}”`,
      })),
    ],
    texts: blocks.flatMap((block) =>
      block.elements
        .filter((element) => element.type === "prose" && !element.fromFile)
        .map((element) => {
          const named = elementLabel(element, {
            elements: block.elements.length,
            title: block.title,
          });
          return {
            elementId: element.id,
            blockId: block.id,
            label: named ? `${named} in “${block.title}”` : `“${block.title}”`,
          };
        }),
    ),
  };
}

const MONTH = new Intl.DateTimeFormat("en-US", { month: "short" });

export function importLabel(held: ShelfImport): string {
  if (held.source === "readme") return "From README";
  const made = new Date(held.createdAt);
  const day = `Pasted ${made.getDate()} ${MONTH.format(made)}`;
  const title = held.title.trim();
  return title ? `${day} · ${title}` : day;
}

export function importCounts(pieces: ShelfPiece[], joiner = " · "): string {
  const sections = pieces.filter((piece) => piece.kind === "section").length;
  const pictures = pieces.length - sections;
  return [counted(sections, "section"), counted(pictures, "picture")]
    .filter(Boolean)
    .join(joiner);
}

function counted(count: number, noun: string): string {
  if (count === 0) return "";
  return `${count} ${noun}${count === 1 ? "" : "s"}`;
}

/** positionAt reads a pointer over a block as before it in its upper half and after it in its lower half. */
export function positionAt(
  index: number,
  pointerY: number,
  rect: { top: number; height: number },
): number {
  return pointerY < rect.top + rect.height / 2 ? index : index + 1;
}

export function withGhost<T>(list: T[], position: number | null, ghost: T) {
  if (position === null) return list;
  return [...list.slice(0, position), ghost, ...list.slice(position)];
}
