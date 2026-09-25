"use client";

import { motion, useReducedMotion } from "framer-motion";
import { CornerDownRight, Plus } from "lucide-react";
import type { ReactNode } from "react";
import { RichText } from "@/components/ui/RichText";
import type { ShelfPiece } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { sectionName } from "./ShelfPiece";
import { useShelf } from "./shelf";
import { useWorkspace } from "./state";

const OPEN = { type: "spring", stiffness: 360, damping: 32 } as const;

/** ShelfRoom moves the page clear of the shelf on wide screens, so every block stays in reach of a drag. */
export function ShelfRoom({ children }: { children: ReactNode }) {
  const workspace = useWorkspace();
  const open = workspace.editing && workspace.pane?.kind === "shelf";
  return (
    <div
      className={cn(
        "transition-[padding] duration-[440ms] ease-[var(--ease-wipe)] motion-reduce:transition-none",
        open && "lg:pr-[28rem]",
      )}
    >
      {children}
    </div>
  );
}

/** GhostBlock is the gap that opens where a dragged section becomes a new block. */
export function GhostBlock({ piece }: { piece: ShelfPiece }) {
  const shelf = useShelf();
  const still = useReducedMotion();
  const placing = shelf.pending !== null;

  return (
    <motion.div
      animate={{ opacity: 1, scale: 1, filter: "blur(0px)" }}
      className="relative flex min-h-40 flex-col gap-3 overflow-hidden rounded-plate border-2 border-accent/70 border-dashed bg-accent-wash/40 px-6 py-5"
      data-shelf-ghost
      initial={still ? false : { opacity: 0, scale: 0.97, filter: "blur(4px)" }}
      transition={OPEN}
    >
      <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <p className="font-display text-title font-medium tracking-tight text-ink wrap-anywhere">
          {sectionName(piece)}
        </p>
        <span
          className={cn(
            "inline-flex items-center gap-1.5 rounded-full bg-action px-3 py-1 text-label font-medium text-on-accent",
            placing && "motion-safe:animate-pulse",
          )}
        >
          <Plus aria-hidden="true" className="size-3.5" />
          {placing ? "Placing…" : "New block"}
        </span>
      </div>
      <RichText
        className="max-h-24 overflow-hidden text-meta text-mute [mask-image:linear-gradient(to_bottom,black_35%,transparent)]"
        text={(piece.text ?? "").slice(0, 420)}
      />
    </motion.div>
  );
}

/** TextTarget marks a text element a dragged section can be added to the end of. */
export function TextTarget({
  blockId,
  children,
  elementId,
}: {
  blockId: string;
  children: ReactNode;
  elementId: string;
}) {
  const shelf = useShelf();
  const aimed = (target: typeof shelf.target) =>
    target !== null && "elementId" in target && target.elementId === elementId;
  const over = shelf.dragging !== null && aimed(shelf.target);
  const adding = shelf.pending !== null && aimed(shelf.pending.target);
  const shown = shelf.dragging !== null || adding;

  return (
    <div
      className={cn(
        "relative",
        "after:pointer-events-none after:absolute after:-inset-3 after:rounded-plate after:opacity-0 after:transition-[opacity,background-color,box-shadow] after:duration-200 after:content-[''] motion-reduce:after:transition-none",
        shown &&
          "after:opacity-100 after:inset-ring-2 after:inset-ring-accent/30",
        (over || adding) && "after:bg-accent-wash/50 after:inset-ring-accent",
      )}
      data-shelf-block={blockId}
      data-shelf-text={elementId}
    >
      {children}
      {over || adding ? (
        <motion.span
          animate={{ opacity: 1, y: 0 }}
          className={cn(
            "pointer-events-none absolute right-0 -bottom-7 z-10 inline-flex items-center gap-1.5 rounded-full bg-action px-3 py-1 text-label font-medium whitespace-nowrap text-on-accent shadow-popover",
            adding && "motion-safe:animate-pulse",
          )}
          initial={{ opacity: 0, y: 4 }}
        >
          <CornerDownRight aria-hidden="true" className="size-3.5" />
          {adding ? "Adding…" : "Add to the end"}
        </motion.span>
      ) : null}
    </div>
  );
}
