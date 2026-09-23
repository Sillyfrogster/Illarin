"use client";

import {
  Columns3,
  Ellipsis,
  EyeOff,
  GripVertical,
  LayoutGrid,
  Trash2,
} from "lucide-react";
import type { DragEventHandler } from "react";
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
import type { WorkBlock } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import {
  BLOCK_WIDTHS,
  type BlockLayout,
  type BlockWidth,
  LAYOUT_LABELS,
  LAYOUTS,
  layoutChoiceIssue,
  WIDTH_FLOORS_PX,
  WIDTH_LABELS,
  widthChoiceIssue,
} from "@/lib/page-arrangement";
import { gripId } from "./arrangement";
import { blockDestinations, relaidBlock } from "./composition";
import { useWorkspace } from "./state";

const TOOL =
  "inline-flex min-h-9 shrink-0 items-center gap-1.5 rounded-control px-2 text-meta font-medium text-mute outline-offset-3 hover:bg-plane hover:text-ink data-[state=open]:bg-plane data-[state=open]:text-ink";

const WIDTH_HINTS: Record<BlockWidth, string> = {
  full: "All twelve columns.",
  half: `Six columns, or Full under ${WIDTH_FLOORS_PX.half}px.`,
  third: `Four columns, then Half or Full under ${WIDTH_FLOORS_PX.third}px.`,
  two_thirds: `Eight columns, or Full under ${WIDTH_FLOORS_PX.two_thirds}px.`,
};

const LAYOUT_HINTS: Record<BlockLayout, string> = {
  duo: "Two equal areas side by side",
  "main-aside": "One wide area and one narrow area",
  single: "One content area",
  "stack-2": "Two content areas in reading order",
  "stack-3": "Three content areas in reading order",
  trio: "Three equal areas side by side",
};

export type BlockGrip = {
  draggable: true;
  onDragEnd: DragEventHandler;
  onDragStart: DragEventHandler;
};

export function BlockTools({
  block,
  grip,
  position,
  suggestedWidth,
  total,
}: {
  block: WorkBlock;
  grip: BlockGrip;
  position: number;
  suggestedWidth?: BlockWidth;
  total: number;
}) {
  const workspace = useWorkspace();
  const { arrangement } = workspace;
  const suggestion =
    suggestedWidth &&
    suggestedWidth !== block.width &&
    !widthChoiceIssue(block.layout, suggestedWidth)
      ? suggestedWidth
      : null;

  function move(to: number) {
    arrangement.move(block.id, to);
    workspace.say(`“${block.title}” moved to ${to + 1} of ${total}.`);
  }

  return (
    <div
      aria-label={`${block.title} controls`}
      className="flex shrink-0 items-center gap-0.5 rounded-plate bg-deep p-0.5 opacity-0 transition-opacity duration-200 group-focus-within/block:opacity-100 group-hover/block:opacity-100 motion-reduce:transition-none max-md:opacity-100"
      data-measurement-ignore
      role="toolbar"
    >
      <button
        aria-label={`Move “${block.title}”, ${position + 1} of ${total}. Drag it, or press the up and down arrows.`}
        className={cn(TOOL, "cursor-grab px-1.5 active:cursor-grabbing")}
        disabled={total < 2 || arrangement.busy}
        id={gripId(block.id)}
        onKeyDown={(event) => {
          if (event.key === "ArrowUp" && position > 0) {
            event.preventDefault();
            move(position - 1);
          }
          if (event.key === "ArrowDown" && position < total - 1) {
            event.preventDefault();
            move(position + 1);
          }
        }}
        type="button"
        {...grip}
      >
        <GripVertical aria-hidden="true" size={15} />
      </button>

      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label={`Width of “${block.title}”, now ${WIDTH_LABELS[block.width]}`}
          className={TOOL}
        >
          <Columns3 aria-hidden="true" size={15} />
          <span className="max-lg:sr-only">{WIDTH_LABELS[block.width]}</span>
          {suggestion ? (
            <span
              aria-hidden="true"
              className="size-1.5 shrink-0 rounded-full bg-accent"
            />
          ) : null}
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>
            <strong className="block text-ui font-medium text-ink">
              Block width
            </strong>
            <span className="mt-1 block text-meta text-mute">
              Choose how much of the page this block occupies.
            </span>
          </DropdownMenuLabel>
          <DropdownMenuRadioGroup
            onValueChange={(width) =>
              workspace.writeBlock({ ...block, width: width as BlockWidth })
            }
            value={block.width}
          >
            {BLOCK_WIDTHS.map((choice) => {
              const issue = widthChoiceIssue(block.layout, choice);
              return (
                <DropdownMenuRadioItem
                  className="items-start py-2.5"
                  disabled={Boolean(issue)}
                  key={choice}
                  value={choice}
                >
                  <Choice
                    hint={issue ?? WIDTH_HINTS[choice]}
                    label={WIDTH_LABELS[choice]}
                    note={choice === suggestion ? "Suggested" : undefined}
                  />
                </DropdownMenuRadioItem>
              );
            })}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      {block.allowedLayouts.length > 1 ? (
        <DropdownMenu>
          <DropdownMenuTrigger
            aria-label={`Layout of “${block.title}”, now ${LAYOUT_LABELS[block.layout]}`}
            className={TOOL}
          >
            <LayoutGrid aria-hidden="true" size={15} />
            <span className="max-lg:sr-only">
              {LAYOUT_LABELS[block.layout]}
            </span>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>
              <strong className="block text-ui font-medium text-ink">
                Content layout
              </strong>
              <span className="mt-1 block text-meta text-mute">
                Arrange the content areas inside this block.
              </span>
            </DropdownMenuLabel>
            <DropdownMenuRadioGroup
              onValueChange={(layout) =>
                workspace.writeBlock(relaidBlock(block, layout as BlockLayout))
              }
              value={block.layout}
            >
              {block.allowedLayouts.map((choice) => {
                const issue = layoutChoiceIssue(
                  choice,
                  block.width,
                  block.elements.map((element) => element.label || "Content"),
                );
                return (
                  <DropdownMenuRadioItem
                    className="items-start py-2.5"
                    disabled={Boolean(issue)}
                    key={choice}
                    value={choice}
                  >
                    <LayoutGlyph layout={choice} />
                    <Choice
                      hint={issue ?? LAYOUT_HINTS[choice]}
                      label={LAYOUT_LABELS[choice]}
                    />
                  </DropdownMenuRadioItem>
                );
              })}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : null}

      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label={`More for “${block.title}”`}
          className={cn(TOOL, "px-1.5")}
        >
          <Ellipsis aria-hidden="true" size={15} />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            disabled={position === 0 || arrangement.busy}
            onSelect={() => move(position - 1)}
          >
            Move earlier
          </DropdownMenuItem>
          <DropdownMenuItem
            disabled={position === total - 1 || arrangement.busy}
            onSelect={() => move(position + 1)}
          >
            Move later
          </DropdownMenuItem>
          {block.hideable ? (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                disabled={arrangement.busy}
                onSelect={() => arrangement.setHidden(block.id, !block.hidden)}
              >
                <EyeOff />
                {block.hidden ? "Show on the public page" : "Hide from readers"}
              </DropdownMenuItem>
            </>
          ) : null}
          {block.required ? null : (
            <>
              {block.hideable ? null : <DropdownMenuSeparator />}
              <DropdownMenuItem
                className="text-stop data-[highlighted]:bg-stop-wash data-[highlighted]:text-stop"
                onSelect={() =>
                  workspace.openPane({ blockId: block.id, kind: "remove" })
                }
              >
                <Trash2 />
                Remove this block
              </DropdownMenuItem>
            </>
          )}
          {total > 1 ? (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuLabel className="text-label text-mute">
                Move it to
              </DropdownMenuLabel>
              {blockDestinations(workspace.blocks, block.id).map(
                (destination) => (
                  <DropdownMenuItem
                    disabled={arrangement.busy}
                    key={destination.position}
                    onSelect={() => move(destination.position)}
                  >
                    {destination.label}
                  </DropdownMenuItem>
                ),
              )}
            </>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

function Choice({
  hint,
  label,
  note,
}: {
  hint: string;
  label: string;
  note?: string;
}) {
  return (
    <span className="min-w-0 flex-1">
      <span className="flex items-baseline gap-2">
        {label}
        {note ? <span className="text-label text-accent">{note}</span> : null}
      </span>
      <span className="mt-0.5 block text-meta text-mute">{hint}</span>
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
      className={cn("mt-0.5 grid !size-5 shrink-0 gap-0.5", GLYPH[layout])}
    >
      {LAYOUTS[layout].slots.map((slot) => (
        <span className="rounded-[2px] bg-current opacity-45" key={slot} />
      ))}
    </span>
  );
}
