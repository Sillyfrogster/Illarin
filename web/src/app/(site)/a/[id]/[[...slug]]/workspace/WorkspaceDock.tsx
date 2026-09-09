"use client";

import { motion, useReducedMotion } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import { BookOpen, Command, LockKeyhole } from "lucide-react";
import { cn } from "@/lib/cn";
import type { SaveState } from "./state";
import { useWorkspace } from "./state";

const STATUS: Record<SaveState, string> = {
  failed: "Not saved",
  private: "Saved privately",
  published: "Readers have this",
  saving: "Saving",
  unsaved: "Unsaved",
};

const TOOL =
  "relative inline-flex size-11 shrink-0 items-center justify-center rounded-control text-field opacity-70 outline-offset-3 hover:bg-field/15 hover:opacity-100 focus-visible:opacity-100 aria-pressed:bg-field/15 aria-pressed:opacity-100";

/** Private work is a ring and published work a filled dot, so the two never read alike. */
function StatusLight({ state }: { state: SaveState }) {
  const reduced = useReducedMotion();
  return (
    <span className="relative flex size-2.5 shrink-0">
      {state === "unsaved" && !reduced ? (
        <motion.span
          animate={{ opacity: [0.5, 0], scale: [1, 2.4] }}
          className="absolute inset-0 rounded-full bg-accent"
          transition={{ duration: 2, repeat: Number.POSITIVE_INFINITY }}
        />
      ) : null}
      <span
        className={cn(
          "relative size-2.5 rounded-full",
          state === "failed" && "bg-stop",
          state === "unsaved" && "bg-accent",
          state === "saving" && "bg-accent",
          state === "private" && "shadow-[inset_0_0_0_2px_var(--color-accent)]",
          state === "published" && "bg-field/60",
        )}
      />
    </span>
  );
}

function Tool({
  active,
  icon: Icon,
  label,
  onClick,
}: {
  active?: boolean;
  icon: LucideIcon;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      aria-label={label}
      aria-pressed={active}
      className={cn(TOOL, "group")}
      onClick={onClick}
      title={label}
      type="button"
    >
      <Icon aria-hidden="true" size={18} />
    </button>
  );
}

/** One bar carries save status, the reading view, the tools and the publication action. */
export function WorkspaceDock({
  detail,
  onJump,
  publicationLabel,
}: {
  detail: string;
  onJump: () => void;
  publicationLabel: string;
}) {
  const workspace = useWorkspace();
  const reduced = useReducedMotion();
  const shifted = workspace.pane !== null;

  return (
    <div
      className={cn(
        "pointer-events-none fixed inset-x-0 bottom-0 z-30 flex justify-center px-3 pb-4 transition-[padding] duration-500 ease-wipe motion-reduce:transition-none md:pb-7",
        shifted && "max-lg:hidden lg:pr-[28rem]",
      )}
    >
      <motion.div
        animate={{ opacity: 1, y: 0 }}
        className="pointer-events-auto flex w-full max-w-[50rem] flex-wrap items-center gap-1 rounded-plate bg-ink p-2 shadow-popover sm:flex-nowrap md:gap-2"
        initial={reduced ? { opacity: 0 } : { opacity: 0, y: 96 }}
        transition={{ duration: reduced ? 0 : 0.45, ease: [0.22, 1, 0.36, 1] }}
      >
        <div
          aria-live="polite"
          className="flex w-full min-w-0 items-center gap-2.5 px-2 text-field sm:w-auto sm:pr-1 sm:pl-3"
        >
          <StatusLight state={workspace.saveState} />
          <span className="min-w-0 leading-4">
            <span className="block truncate text-meta font-medium">
              {STATUS[workspace.saveState]}
            </span>
            <span className="block truncate text-label opacity-65">
              {detail}
            </span>
          </span>
        </div>

        <div className="flex min-w-0 flex-1 items-center gap-0.5 sm:justify-center">
          <Tool
            icon={BookOpen}
            label="Reading view"
            onClick={workspace.stopEditing}
          />
          <Tool icon={Command} label="Go to content" onClick={onJump} />
          <Tool
            active={workspace.pane?.kind === "access"}
            icon={LockKeyhole}
            label="Access and published state"
            onClick={() => workspace.openPane({ kind: "access" })}
          />
        </div>

        <div className="flex shrink-0 items-center gap-1">
          <button
            className="inline-flex min-h-11 items-center rounded-control px-3.5 text-ui font-medium text-field opacity-75 outline-offset-3 hover:bg-field/15 hover:opacity-100 disabled:opacity-35"
            disabled={workspace.busy || !workspace.dirty}
            onClick={workspace.save}
            type="button"
          >
            {workspace.busy ? "Saving…" : "Save"}
          </button>
          <button
            className="inline-flex min-h-11 shrink-0 items-center rounded-control bg-field px-5 text-ui font-medium text-ink outline-offset-3 hover:opacity-90 disabled:opacity-35"
            disabled={workspace.busy}
            onClick={() => workspace.openPane({ kind: "publication" })}
            type="button"
          >
            {publicationLabel}
          </button>
        </div>
      </motion.div>
    </div>
  );
}
