"use client";

import { Ellipsis, EyeOff, GripVertical, Plus, Trash2 } from "lucide-react";
import { type CSSProperties, type KeyboardEvent, useState } from "react";
import { createPortal } from "react-dom";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Tooltip } from "@/components/ui/tooltip";
import type { WorkBlock } from "@/lib/api/query";
import { cn, focusRing } from "@/lib/cn";
import {
  BLOCK_WIDTHS,
  type BlockLayout,
  type BlockWidth,
  LAYOUT_LABELS,
  LAYOUTS,
  layoutChoiceIssue,
  WIDTH_COLUMNS,
  WIDTH_LABELS,
  widthChoiceIssue,
} from "@/lib/page-arrangement";
import { AddBlock } from "./AddBlock";
import { relaidBlock } from "./composition";
import { useWorkspace } from "./state";
import type { Arranging } from "./use-arranging";

/** Undoes the arrange map's zoom so a control keeps its real size on a shrunken block. */
const UNZOOM = { zoom: "calc(1 / var(--map-zoom, 1))" } as CSSProperties;

/** BlockMenu holds everything a block can change besides its place: width, layout, hiding and removing. */
export function BlockMenu({ block }: { block: WorkBlock }) {
  const workspace = useWorkspace();
  const { arrangement } = workspace;
  return (
    <DropdownMenu>
      <Tooltip content="Block options">
        <DropdownMenuTrigger asChild>
          <Button
            aria-label={`Options for “${block.title}”`}
            className="bg-field shadow-none ring-1 ring-rule hover:bg-fill"
            size="icon-compact"
            variant="ghost"
          >
            <Ellipsis aria-hidden="true" />
          </Button>
        </DropdownMenuTrigger>
      </Tooltip>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuLabel className="text-label text-mute">
          Width
        </DropdownMenuLabel>
        <DropdownMenuRadioGroup
          onValueChange={(width) =>
            workspace.writeBlock({ ...block, width: width as BlockWidth })
          }
          value={block.width}
        >
          {BLOCK_WIDTHS.map((choice) => (
            <DropdownMenuRadioItem
              disabled={Boolean(widthChoiceIssue(block.layout, choice))}
              key={choice}
              value={choice}
            >
              <WidthGlyph width={choice} />
              {WIDTH_LABELS[choice]}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        {block.allowedLayouts.length > 1 ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuLabel className="text-label text-mute">
              Inside the block
            </DropdownMenuLabel>
            <DropdownMenuRadioGroup
              onValueChange={(layout) =>
                workspace.writeBlock(relaidBlock(block, layout as BlockLayout))
              }
              value={block.layout}
            >
              {block.allowedLayouts.map((choice) => (
                <DropdownMenuRadioItem
                  disabled={Boolean(
                    layoutChoiceIssue(
                      choice,
                      block.width,
                      block.elements.map((one) => one.label || "Content"),
                    ),
                  )}
                  key={choice}
                  value={choice}
                >
                  <LayoutGlyph layout={choice} />
                  {LAYOUT_LABELS[choice]}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </>
        ) : null}
        {block.hideable || !block.required ? <DropdownMenuSeparator /> : null}
        {block.hideable ? (
          <DropdownMenuItem
            disabled={arrangement.busy}
            onSelect={() => arrangement.setHidden(block.id, !block.hidden)}
          >
            <EyeOff />
            {block.hidden ? "Show to readers" : "Hide from readers"}
          </DropdownMenuItem>
        ) : null}
        {block.required ? null : (
          <DropdownMenuItem
            className="text-stop data-[highlighted]:bg-stop-wash data-[highlighted]:text-stop"
            onSelect={() =>
              workspace.openPane({ blockId: block.id, kind: "remove" })
            }
          >
            <Trash2 />
            Remove block
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** useBlockKeys moves a block with the up and down arrows and changes its width with left and right, the keyboard's way to do what dragging does. */
function useBlockKeys(block: WorkBlock, position: number, total: number) {
  const workspace = useWorkspace();
  const allowed = BLOCK_WIDTHS.filter(
    (choice) => !widthChoiceIssue(block.layout, choice),
  ).sort((a, b) => WIDTH_COLUMNS[a] - WIDTH_COLUMNS[b]);
  return (event: KeyboardEvent) => {
    const move = (to: number) => {
      event.preventDefault();
      workspace.arrangement.move(block.id, to);
      workspace.say(`“${block.title}” moved to ${to + 1} of ${total}.`);
    };
    if (event.key === "ArrowUp" && position > 0) move(position - 1);
    if (event.key === "ArrowDown" && position < total - 1) move(position + 1);
    if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
      const at = allowed.indexOf(block.width);
      const next = allowed[at + (event.key === "ArrowRight" ? 1 : -1)];
      if (!next) return;
      event.preventDefault();
      workspace.writeBlock({ ...block, width: next });
      workspace.say(
        `“${block.title}” is now ${WIDTH_LABELS[next].toLowerCase()} width.`,
      );
    }
  };
}

const KEYS_HINT = "Arrow keys move it and change its width.";

/** MapTile is a block on the arrange map: press and drag it to move it, drag its right edge to resize it, click it to write in it. */
export function MapTile({
  arranging,
  block,
  onOpen,
  position,
  startColumn,
  suggestion,
  total,
}: {
  arranging: Arranging;
  block: WorkBlock;
  onOpen: () => void;
  position: number;
  startColumn: number;
  suggestion?: BlockWidth;
  total: number;
}) {
  const keys = useBlockKeys(block, position, total);
  const stretching = arranging.stretch?.blockId === block.id;
  return (
    <>
      <button
        aria-label={`${block.title}, ${WIDTH_LABELS[block.width].toLowerCase()} width, ${position + 1} of ${total}. Enter writes in it. ${KEYS_HINT}`}
        className={cn(
          "absolute inset-0 z-10 cursor-grab rounded-card active:cursor-grabbing",
          focusRing,
          "focus-visible:ring-2",
        )}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            onOpen();
            return;
          }
          keys(event);
        }}
        onPointerDown={(event) => arranging.pressBlock(event, block, onOpen)}
        type="button"
      />
      <div
        className="pointer-events-none absolute top-3 left-3 z-20 flex items-center gap-1.5"
        style={UNZOOM}
      >
        <span className="inline-flex h-7 items-center gap-1 rounded-chip bg-field/90 px-2 text-label font-medium text-ink ring-1 ring-ink/8 tabular-nums">
          <WidthGlyph width={block.width} />
          {WIDTH_LABELS[block.width]}
        </span>
        {block.hidden ? (
          <span className="inline-flex h-7 items-center gap-1 rounded-chip bg-field/90 px-2 text-label text-mute ring-1 ring-ink/8">
            <EyeOff aria-hidden="true" className="size-3.5" />
            Hidden
          </span>
        ) : null}
        <Suggestion block={block} width={suggestion} />
      </div>
      <div
        className="absolute top-3 right-3 z-20 opacity-0 transition-opacity duration-150 group-focus-within/block:opacity-100 group-hover/block:opacity-100 pointer-coarse:opacity-100"
        style={UNZOOM}
      >
        <BlockMenu block={block} />
      </div>
      <Edge
        arranging={arranging}
        block={block}
        startColumn={startColumn}
        stretching={stretching}
        unzoom
      />
    </>
  );
}

/** Frame is a block's handles on the page itself: a tab to drag it by, its options, and a right edge to drag wider or narrower. */
export function Frame({
  arranging,
  block,
  position,
  startColumn,
  suggestion,
  total,
}: {
  arranging: Arranging;
  block: WorkBlock;
  position: number;
  startColumn: number;
  suggestion?: BlockWidth;
  total: number;
}) {
  const keys = useBlockKeys(block, position, total);
  const stretching = arranging.stretch?.blockId === block.id;
  return (
    <>
      <div className="absolute -top-4 left-3 z-20 flex -translate-y-1/2 items-center gap-1">
        <button
          aria-label={`Move “${block.title}”, ${position + 1} of ${total}. ${KEYS_HINT}`}
          className={cn(
            "inline-flex h-8 cursor-grab touch-none items-center gap-1.5 rounded-control bg-field pr-2.5 pl-1.5 text-label font-medium text-mute ring-1 ring-rule transition-colors duration-150 hover:text-ink hover:ring-accent/60 active:cursor-grabbing group-hover/block:text-ink",
            focusRing,
          )}
          onKeyDown={keys}
          onPointerDown={(event) => arranging.pressBlock(event, block)}
          type="button"
        >
          <GripVertical aria-hidden="true" className="size-4" />
          <WidthGlyph width={block.width} />
          <span className="tabular-nums">{WIDTH_LABELS[block.width]}</span>
          {block.hidden ? (
            <>
              <span aria-hidden="true">·</span>
              <EyeOff aria-hidden="true" className="size-3.5" />
              Hidden
            </>
          ) : null}
        </button>
        <Suggestion block={block} width={suggestion} />
      </div>
      <div className="absolute -top-4 right-3 z-20 -translate-y-1/2">
        <BlockMenu block={block} />
      </div>
      <Edge
        arranging={arranging}
        block={block}
        startColumn={startColumn}
        stretching={stretching}
      />
    </>
  );
}

/** Suggestion offers the width that suits a block's length, as a one-press change beside the block's own width. */
function Suggestion({
  block,
  width,
}: {
  block: WorkBlock;
  width?: BlockWidth;
}) {
  const workspace = useWorkspace();
  if (!width || width === block.width || widthChoiceIssue(block.layout, width))
    return null;
  const narrower = WIDTH_COLUMNS[width] < WIDTH_COLUMNS[block.width];
  const label = WIDTH_LABELS[width].toLowerCase();
  return (
    <Tooltip
      content={
        narrower
          ? `It is short enough for ${label} width.`
          : `It reads shorter at ${label} width.`
      }
    >
      <button
        className={cn(
          "pointer-events-auto inline-flex h-8 items-center gap-1.5 rounded-control bg-field px-2.5 text-label font-medium text-mute ring-1 ring-rule max-md:hidden transition-colors duration-150 hover:text-ink hover:ring-accent/60 [&>span:first-child]:text-accent",
          focusRing,
        )}
        onClick={() => {
          workspace.writeBlock({ ...block, width });
          workspace.say(`“${block.title}” is now ${label} width.`);
        }}
        type="button"
      >
        <WidthGlyph width={width} />
        {narrower ? `Fits ${label}` : `Try ${label}`}
      </button>
    </Tooltip>
  );
}

function Edge({
  arranging,
  block,
  startColumn,
  stretching,
  unzoom = false,
}: {
  arranging: Arranging;
  block: WorkBlock;
  startColumn: number;
  stretching: boolean;
  unzoom?: boolean;
}) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "group/edge absolute inset-y-0 z-20 hidden w-5 cursor-ew-resize touch-none justify-center md:flex",
        unzoom ? "-right-2.5" : "-right-[1.125rem]",
      )}
      onPointerDown={(event) => arranging.pressEdge(event, block, startColumn)}
    >
      <span
        className={cn(
          "sticky top-[calc(50vh-1.5rem)] my-6 flex h-12 w-1.5 items-center rounded-full bg-rule transition-[background-color,width] duration-150 group-hover/edge:w-2 group-hover/edge:bg-accent",
          stretching && "w-2 bg-accent",
        )}
        style={unzoom ? UNZOOM : undefined}
      >
        {stretching ? (
          <span className="absolute left-4 inline-flex h-8 items-center gap-1.5 rounded-control bg-ink px-2.5 text-meta font-medium whitespace-nowrap text-field shadow-popover">
            <WidthGlyph width={block.width} />
            {WIDTH_LABELS[block.width]}
          </span>
        ) : null}
      </span>
    </div>
  );
}

const COLUMNS = Array.from({ length: 12 }, (_, column) => `column-${column}`);

/** ColumnGuides shows the page's twelve columns while a block is being resized, so the snap points are visible. */
export function ColumnGuides({ shown }: { shown: boolean }) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "pointer-events-none absolute inset-0 -z-1 hidden grid-cols-12 gap-x-[var(--block-grid-gap)] transition-opacity duration-200 md:grid",
        shown ? "opacity-100" : "opacity-0",
      )}
    >
      {COLUMNS.map((column) => (
        <span className="rounded-chip bg-accent-wash/70" key={column} />
      ))}
    </div>
  );
}

/** LiftCard rides under the pointer while a block is being moved, naming what is held. */
export function LiftCard({ arranging }: { arranging: Arranging }) {
  if (!arranging.lift) return null;
  return createPortal(
    <div
      aria-hidden="true"
      className="pointer-events-none fixed top-0 left-0 z-90 flex h-11 -rotate-2 items-center gap-2 rounded-control bg-plane pr-4 pl-2.5 text-ui font-medium text-ink shadow-popover ring-1 ring-accent"
      ref={arranging.liftCard}
    >
      <GripVertical aria-hidden="true" className="size-4 text-accent" />
      <span className="max-w-64 truncate">{arranging.lift.title}</span>
      <span className="text-meta font-normal text-mute">
        {WIDTH_LABELS[arranging.lift.width]}
      </span>
    </div>,
    document.body,
  );
}

/** Seam is the line between two rows of blocks where a new block can be added in place. */
export function Seam({ at, last = false }: { at: number; last?: boolean }) {
  const workspace = useWorkspace();
  const [open, setOpen] = useState(false);
  if (workspace.addableBlocks.length === 0) return null;
  return (
    <div
      className={cn(
        "group/seam pointer-events-none relative col-span-full flex h-0",
        !last && "max-md:hidden",
        "items-center justify-center md:[grid-column:1/-1] md:[grid-row:var(--seam-row)]",
        last
          ? "md:self-end md:translate-y-[calc(var(--seam-gap)/2)]"
          : "md:self-start md:-translate-y-[calc(var(--seam-gap)/2)]",
      )}
    >
      <span
        aria-hidden="true"
        className={cn(
          "absolute inset-x-0 h-px bg-accent opacity-0 transition-opacity duration-150 group-hover/seam:opacity-60",
          open && "opacity-60",
        )}
      />
      <Popover onOpenChange={setOpen} open={open}>
        <PopoverTrigger asChild>
          <button
            aria-label={
              last
                ? "Add a block at the end"
                : `Add a block here, before block ${at + 1}`
            }
            className={cn(
              "pointer-events-auto relative inline-flex h-9 items-center gap-1.5 rounded-control bg-field px-3 text-meta font-medium text-mute ring-1 ring-rule transition-[color,opacity,box-shadow] duration-150 hover:text-ink hover:ring-accent",
              "opacity-0 group-hover/seam:opacity-100 focus-visible:opacity-100 pointer-coarse:opacity-100",
              (open || last) && "opacity-100",
              focusRing,
            )}
            type="button"
          >
            <Plus aria-hidden="true" className="size-4" />
            Add a block
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-[min(26rem,calc(100vw-2rem))] p-2">
          <AddBlock at={at} onDone={() => setOpen(false)} />
        </PopoverContent>
      </Popover>
    </div>
  );
}

/** WidthGlyph draws a width as the share of a row it fills. */
export function WidthGlyph({ width }: { width: BlockWidth }) {
  return (
    <span
      aria-hidden="true"
      className="inline-grid h-2.5 w-5 shrink-0 grid-cols-12 gap-px rounded-[2px] bg-current/20 p-px"
    >
      <span
        className="rounded-[1px] bg-current"
        style={{ gridColumn: `span ${WIDTH_COLUMNS[width]}` }}
      />
    </span>
  );
}

const GLYPH: Record<BlockLayout, string> = {
  duo: "grid-cols-2",
  "main-aside": "[grid-template-columns:2fr_1fr]",
  single: "grid-cols-1",
  "stack-2": "grid-cols-1",
  "stack-3": "grid-cols-1",
  trio: "grid-cols-3",
};

function LayoutGlyph({ layout }: { layout: BlockLayout }) {
  return (
    <span
      aria-hidden="true"
      className={cn("grid !size-4 shrink-0 gap-px", GLYPH[layout])}
    >
      {LAYOUTS[layout].slots.map((slot) => (
        <span className="rounded-[1px] bg-current opacity-45" key={slot} />
      ))}
    </span>
  );
}
