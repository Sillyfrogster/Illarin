"use client";

import { Ellipsis, EyeOff, GripVertical, Trash2 } from "lucide-react";
import type { CSSProperties, KeyboardEvent } from "react";
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
import { relaidBlock } from "./composition";
import { useWorkspace } from "./state";
import type { ArrangeHandlers } from "./use-arranging";

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
  stretching,
  suggestion,
  total,
}: {
  arranging: ArrangeHandlers;
  block: WorkBlock;
  onOpen: () => void;
  position: number;
  startColumn: number;
  stretching: boolean;
  suggestion?: BlockWidth;
  total: number;
}) {
  const keys = useBlockKeys(block, position, total);
  return (
    <>
      <button
        aria-label={`${block.title}, ${WIDTH_LABELS[block.width].toLowerCase()} width, ${position + 1} of ${total}. Enter writes in it. ${KEYS_HINT}`}
        className={cn(
          "absolute inset-0 z-10 cursor-grab touch-pan-y rounded-card active:cursor-grabbing",
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
        <span
          aria-hidden="true"
          className="pointer-events-auto hidden size-9 touch-none items-center justify-center rounded-control bg-field text-mute ring-1 ring-rule pointer-coarse:inline-flex"
          onPointerDown={(event) => arranging.pressGrip(event, block)}
        >
          <GripVertical className="size-4.5" />
        </span>
        <span className="inline-flex h-7 items-center gap-1.5 rounded-chip bg-field px-2 text-label font-medium text-ink ring-1 ring-ink/8 tabular-nums max-md:hidden">
          <WidthGlyph width={block.width} />
          {WIDTH_LABELS[block.width]}
        </span>
        {block.hidden ? (
          <span className="inline-flex h-7 items-center gap-1 rounded-chip bg-field px-2 text-label text-mute ring-1 ring-ink/8">
            <EyeOff aria-hidden="true" className="size-3.5" />
            Hidden
          </span>
        ) : null}
        {stretching ? null : <Suggestion block={block} width={suggestion} />}
      </div>
      <div
        className="absolute top-3 right-3 z-20 opacity-0 transition-opacity duration-150 group-focus-within/block:opacity-100 group-hover/block:opacity-100 pointer-coarse:opacity-100"
        style={UNZOOM}
      >
        <BlockMenu block={block} />
      </div>
      {stretching ? <WidthReadout width={block.width} /> : null}
      <Edge
        arranging={arranging}
        block={block}
        startColumn={startColumn}
        stretching={stretching}
      />
    </>
  );
}

/** WidthReadout names the width a block will take while its edge is held, large enough to read without looking away from the edge. */
function WidthReadout({ width }: { width: BlockWidth }) {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 z-30 flex items-center justify-center rounded-card bg-accent-wash/80"
    >
      <span
        className="inline-flex animate-pop items-center gap-3 rounded-control bg-action px-5 py-3 text-title font-medium text-on-accent shadow-popover"
        style={UNZOOM}
      >
        <WidthGlyph width={width} />
        {WIDTH_LABELS[width]}
      </span>
    </div>
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
}: {
  arranging: ArrangeHandlers;
  block: WorkBlock;
  startColumn: number;
  stretching: boolean;
}) {
  return (
    <div
      aria-hidden="true"
      className="group/edge absolute inset-y-0 -right-6 z-40 hidden w-12 cursor-ew-resize touch-none justify-center md:flex"
      onPointerDown={(event) => arranging.pressEdge(event, block, startColumn)}
    >
      <span
        className={cn(
          "sticky top-[calc(50vh-1.75rem)] my-6 flex h-14 w-4 items-center justify-center gap-0.5 rounded-full bg-field ring-1 ring-edge transition-[background-color,box-shadow,opacity] duration-150 group-hover/edge:bg-action group-hover/edge:ring-action opacity-0 group-hover/block:opacity-100",
          stretching && "bg-action opacity-100 ring-action",
        )}
        style={UNZOOM}
      >
        <span
          className={cn(
            "h-5 w-px rounded-full bg-mute group-hover/edge:bg-on-accent",
            stretching && "bg-on-accent",
          )}
        />
        <span
          className={cn(
            "h-5 w-px rounded-full bg-mute group-hover/edge:bg-on-accent",
            stretching && "bg-on-accent",
          )}
        />
      </span>
    </div>
  );
}

const COLUMNS = Array.from({ length: 12 }, (_, column) => column + 1);

/** ColumnGuides lies the page's twelve columns under the arrange map, and lights the ones a block being resized will fill. */
export function ColumnGuides({
  span,
}: {
  span: { start: number; columns: number } | null;
}) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "pointer-events-none absolute inset-0 -z-1 hidden grid-cols-12 gap-x-[var(--block-grid-gap)] transition-opacity duration-200 md:grid",
        span ? "opacity-100" : "opacity-0",
      )}
    >
      {COLUMNS.map((column) => (
        <span
          className={cn(
            "rounded-chip transition-colors duration-150",
            span && column >= span.start && column < span.start + span.columns
              ? "bg-accent-wash"
              : "bg-ink/5",
          )}
          key={column}
        />
      ))}
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
