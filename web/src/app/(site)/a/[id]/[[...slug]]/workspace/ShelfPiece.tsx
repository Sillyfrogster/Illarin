"use client";

import { useDraggable } from "@dnd-kit/core";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import {
  ArrowUpRight,
  Check,
  ChevronDown,
  Globe,
  GripVertical,
  LoaderCircle,
  Plus,
} from "lucide-react";
import Image from "next/image";
import { useId, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { RichText } from "@/components/ui/RichText";
import type { ShelfPiece } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { useShelf } from "./shelf";
import { placeChoices } from "./shelf-places";
import { useWorkspace } from "./state";

const SPRING = { type: "spring", stiffness: 420, damping: 36 } as const;
const PREVIEW_CHARACTERS = 1400;

export function sectionName(piece: ShelfPiece): string {
  return piece.heading.trim() || "Opening";
}

function wordCount(text = ""): string {
  const words = text.split(/\s+/).filter(Boolean).length;
  return `${words.toLocaleString("en-US")} ${words === 1 ? "word" : "words"}`;
}

/** SectionRow is one section on the shelf's outline: dragged by its row on desktop, opened for a preview, placed from its menu. */
export function SectionRow({
  last,
  piece,
  placedIn,
}: {
  last: boolean;
  piece: ShelfPiece;
  placedIn: string | null | undefined;
}) {
  const shelf = useShelf();
  const still = useReducedMotion();
  const preview = useId();
  const [open, setOpen] = useState(false);
  const placed = placedIn !== undefined;
  const { isDragging, listeners, setNodeRef } = useDraggable({
    id: `piece:${piece.id}`,
    data: { piece },
    disabled: placed || !shelf.canDrag || shelf.busy !== null,
  });
  const busy = shelf.busy === piece.id;

  return (
    <motion.li
      className="relative grid grid-cols-[1.5rem_minmax(0,1fr)] gap-x-2"
      exit={{ opacity: 0, height: 0 }}
      layout={still ? false : "position"}
      transition={SPRING}
    >
      <Spine done={placed} last={last} working={busy} />
      <div
        className={cn(
          "min-w-0 rounded-control transition-colors duration-150 motion-reduce:transition-none",
          !placed && "hover:bg-deep",
          open && !placed && "bg-deep",
          isDragging && "opacity-35",
        )}
      >
        <div className="flex min-h-11 items-center gap-1 pr-1">
          <button
            aria-controls={preview}
            aria-expanded={placed ? undefined : open}
            className={cn(
              "group/row flex min-h-11 min-w-0 flex-1 items-center gap-2 rounded-control py-2 pl-2 text-left outline-offset-2",
              shelf.canDrag &&
                !placed &&
                "cursor-grab touch-none active:cursor-grabbing",
            )}
            disabled={placed}
            onClick={() => setOpen((now) => !now)}
            ref={setNodeRef}
            type="button"
            {...(shelf.canDrag && !placed ? listeners : {})}
          >
            <span
              className={cn(
                "min-w-0 flex-1 truncate font-ui text-ui font-medium",
                placed ? "text-mute" : "text-ink",
              )}
            >
              {sectionName(piece)}
            </span>
            {placed ? null : (
              <>
                <span className="shrink-0 text-label text-mute tabular-nums">
                  {wordCount(piece.text)}
                </span>
                {shelf.canDrag ? (
                  <GripVertical
                    aria-hidden="true"
                    className="size-4 shrink-0 text-mute opacity-0 transition-opacity group-hover/row:opacity-100 group-focus-visible/row:opacity-100"
                  />
                ) : (
                  <ChevronDown
                    aria-hidden="true"
                    className={cn(
                      "size-4 shrink-0 text-mute transition-transform motion-reduce:transition-none",
                      open && "rotate-180",
                    )}
                  />
                )}
              </>
            )}
          </button>
          {placed ? (
            placedIn ? (
              <Button
                className="text-accent"
                onClick={() => shelf.goTo(placedIn)}
                size="compact"
                variant="ghost"
              >
                On the page
                <ArrowUpRight aria-hidden="true" />
              </Button>
            ) : (
              <span className="px-3 text-label text-mute">On the page</span>
            )
          ) : (
            <PlaceMenu piece={piece} />
          )}
        </div>
        <AnimatePresence initial={false}>
          {open && !placed ? (
            <motion.div
              animate={{ height: "auto", opacity: 1 }}
              className="overflow-hidden"
              exit={{ height: 0, opacity: 0 }}
              id={preview}
              initial={{ height: 0, opacity: 0 }}
              transition={still ? { duration: 0 } : SPRING}
            >
              <div className="flex flex-col gap-3 px-3 pt-1 pb-3">
                <RichText
                  className="max-h-60 overflow-hidden rounded-control bg-plane px-4 py-3 text-meta text-ink/85 [mask-image:linear-gradient(to_bottom,black_75%,transparent)] [&_h4]:text-meta"
                  text={(piece.text ?? "").slice(0, PREVIEW_CHARACTERS)}
                />
                <div className="flex justify-end">
                  <LetGo piece={piece} />
                </div>
              </div>
            </motion.div>
          ) : null}
        </AnimatePresence>
      </div>
    </motion.li>
  );
}

/** Spine draws the outline's line and a node per piece, which fills in once the piece is on the page. */
function Spine({
  done,
  last,
  working,
}: {
  done: boolean;
  last: boolean;
  working: boolean;
}) {
  return (
    <span aria-hidden="true" className="relative flex justify-center">
      {last ? null : (
        <span className="absolute top-[2.25rem] -bottom-[1.125rem] w-0.5 rounded-full bg-rule/70" />
      )}
      <motion.span
        animate={{ scale: done ? [1.35, 1] : 1 }}
        className={cn(
          "relative mt-[1.125rem] flex size-3.5 items-center justify-center rounded-full transition-colors duration-300 motion-reduce:transition-none",
          done
            ? "bg-action text-on-accent"
            : "bg-plane inset-ring-2 inset-ring-edge",
        )}
        transition={{ duration: 0.35 }}
      >
        {working ? (
          <LoaderCircle className="size-3 animate-spin text-accent motion-reduce:animate-none" />
        ) : done ? (
          <Check className="size-2.5" strokeWidth={3.5} />
        ) : null}
      </motion.span>
    </span>
  );
}

/** LiftedSection is the section under the pointer while it is dragged. */
export function LiftedSection({ piece }: { piece: ShelfPiece }) {
  const still = useReducedMotion();
  return (
    <motion.div
      animate={still ? {} : { scale: 1.04, rotate: -2 }}
      className="flex w-72 cursor-grabbing items-center gap-3 rounded-plate bg-plane py-3 pr-4 pl-3 shadow-popover inset-ring-2 inset-ring-accent"
      initial={still ? false : { scale: 1, rotate: 0 }}
      transition={{ type: "spring", stiffness: 420, damping: 24 }}
    >
      <span className="flex size-9 shrink-0 items-center justify-center rounded-control bg-action text-on-accent">
        <Plus aria-hidden="true" className="size-4.5" />
      </span>
      <span className="min-w-0">
        <span className="block truncate font-display text-ui font-medium text-ink">
          {sectionName(piece)}
        </span>
        <span className="block text-label text-mute tabular-nums">
          {wordCount(piece.text)}
        </span>
      </span>
    </motion.div>
  );
}

function PlaceMenu({ piece }: { piece: ShelfPiece }) {
  const workspace = useWorkspace();
  const shelf = useShelf();
  const choices = placeChoices(workspace.blocks);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label={`Place “${sectionName(piece)}”`}
          className="inline-flex size-11 shrink-0 items-center justify-center rounded-full text-accent outline-offset-2 hover:bg-accent-wash disabled:opacity-45 data-[state=open]:bg-action data-[state=open]:text-on-accent"
          disabled={shelf.busy !== null}
          type="button"
        >
          <Plus aria-hidden="true" className="size-5" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="max-w-[min(22rem,calc(100vw-2rem))]"
      >
        <DropdownMenuLabel className="text-label text-mute">
          New block
        </DropdownMenuLabel>
        {choices.blocks.map((choice) => (
          <DropdownMenuItem
            key={choice.position}
            onSelect={() =>
              void shelf.place(piece, { position: choice.position })
            }
          >
            <span className="min-w-0 truncate">{choice.label}</span>
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuLabel className="text-label text-mute">
          Add to the end of
        </DropdownMenuLabel>
        {choices.texts.length === 0 ? (
          <DropdownMenuItem disabled>No text on the page yet</DropdownMenuItem>
        ) : (
          choices.texts.map((choice) => (
            <DropdownMenuItem
              key={choice.elementId}
              onSelect={() =>
                void shelf.place(piece, {
                  elementId: choice.elementId,
                  blockId: choice.blockId,
                })
              }
            >
              <span className="min-w-0 truncate">{choice.label}</span>
            </DropdownMenuItem>
          ))
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function LetGo({ piece }: { piece: ShelfPiece }) {
  const shelf = useShelf();
  return (
    <Button
      className="text-mute hover:bg-stop-wash hover:text-stop"
      disabled={shelf.busy !== null}
      onClick={() => shelf.letGo(piece)}
      size="compact"
      variant="ghost"
    >
      Let go
    </Button>
  );
}

function hostOf(address = ""): string {
  try {
    return new URL(address).hostname;
  } catch {
    return address;
  }
}

/** PictureTile is one picture on the shelf, placed into its section's block from its menu. */
export function PictureTile({
  piece,
  placedIn,
}: {
  piece: ShelfPiece;
  placedIn: string | null | undefined;
}) {
  const workspace = useWorkspace();
  const shelf = useShelf();
  const still = useReducedMotion();
  const file = useRef<HTMLInputElement>(null);
  const home = piece.blockId
    ? workspace.blocks.find((block) => block.id === piece.blockId)
    : undefined;
  const section = piece.section.trim();
  const name = piece.name?.trim() || "Untitled picture";
  const placed = placedIn !== undefined;
  const busy = shelf.busy === piece.id;

  return (
    <motion.li
      className="w-28 shrink-0"
      exit={{ opacity: 0, scale: 0.9 }}
      layout={still ? false : "position"}
      transition={SPRING}
    >
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            aria-label={`${name}, from ${section ? `“${section}”` : "the opening"}`}
            className={cn(
              "relative flex size-28 items-center justify-center overflow-hidden rounded-control outline-offset-2",
              piece.media
                ? "bg-media"
                : "border-2 border-rule border-dashed text-mute hover:border-accent hover:text-accent",
            )}
            disabled={placed || shelf.busy !== null}
            type="button"
          >
            {piece.media ? (
              <Image
                alt=""
                className={cn(
                  "size-full object-cover transition-transform duration-300 hover:scale-105 motion-reduce:transition-none",
                  placed && "opacity-40",
                )}
                height={piece.media.height}
                sizes="112px"
                src={piece.media.thumbUrl}
                unoptimized
                width={piece.media.width}
              />
            ) : placed || busy ? null : (
              <span className="flex flex-col items-center gap-1.5 px-2 text-center">
                <Globe aria-hidden="true" className="size-5" />
                <span className="w-full truncate text-label">
                  {hostOf(piece.address)}
                </span>
              </span>
            )}
            {placed || busy ? (
              <span className="absolute inset-0 flex items-center justify-center">
                <span className="flex size-8 items-center justify-center rounded-full bg-action text-on-accent">
                  {busy ? (
                    <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" />
                  ) : (
                    <Check className="size-4" strokeWidth={3} />
                  )}
                </span>
              </span>
            ) : null}
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="max-w-72">
          <DropdownMenuLabel className="text-label text-mute">
            {name}
          </DropdownMenuLabel>
          {piece.media ? (
            <DropdownMenuItem onSelect={() => void shelf.place(piece)}>
              <span className="min-w-0 truncate">
                {home
                  ? `Place in “${home.title}”`
                  : `Place in a new “${section || "Pictures"}” block`}
              </span>
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onSelect={() => file.current?.click()}>
              Upload a copy
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className="text-stop data-[highlighted]:bg-stop-wash data-[highlighted]:text-stop"
            onSelect={() => shelf.letGo(piece)}
          >
            Let go
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <p className="mt-1.5 truncate text-label text-mute">
        {section || "Opening"}
      </p>
      <input
        accept="image/*"
        className="sr-only"
        onChange={(event) => {
          const chosen = event.target.files?.[0];
          if (chosen) void shelf.place(piece, { file: chosen });
          event.target.value = "";
        }}
        ref={file}
        tabIndex={-1}
        type="file"
      />
    </motion.li>
  );
}
