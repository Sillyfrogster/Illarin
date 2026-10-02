"use client";

import { motion, useReducedMotion } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn, focusRing } from "@/lib/cn";

export type DockState =
  | "failed"
  | "unsaved"
  | "saving"
  | "private"
  | "published";

const SAVED: Record<DockState, string> = {
  failed: "Not saved",
  private: "Saved",
  published: "Saved",
  saving: "Saving",
  unsaved: "Saving",
};

/** StatusLight is the save state as a dot: violet and breathing while saving, a stop colour when a save failed, quiet when saved. */
export function StatusLight({ state }: { state: DockState }) {
  const reduced = useReducedMotion();
  const busy = state === "saving" || state === "unsaved";
  return (
    <span className="relative flex size-2.5 shrink-0">
      {busy && !reduced ? (
        <motion.span
          animate={{ opacity: [0.6, 0], scale: [1, 2.4] }}
          className="absolute inset-0 rounded-full bg-accent"
          transition={{ duration: 1.4, repeat: Number.POSITIVE_INFINITY }}
        />
      ) : null}
      <span
        className={cn(
          "relative size-2.5 rounded-full",
          state === "failed" && "bg-stop",
          busy && "bg-accent",
          (state === "private" || state === "published") && "bg-mute/60",
        )}
      />
    </span>
  );
}

/** SaveStatus says whether the creator's edits are safe, and under it what readers see right now. */
export function SaveStatus({
  detail,
  state,
  compact = false,
}: {
  detail: string;
  state: DockState;
  compact?: boolean;
}) {
  return (
    <div
      aria-live="polite"
      className="flex min-w-0 items-center gap-2.5"
      title={compact ? detail : undefined}
    >
      <StatusLight state={state} />
      <span className="min-w-0 leading-4">
        <span
          className={cn(
            "block truncate text-meta font-medium",
            state === "failed" ? "text-stop" : "text-ink",
          )}
        >
          {SAVED[state]}
        </span>
        <span
          className={cn(
            "block truncate text-label text-mute",
            compact && "max-md:sr-only",
          )}
        >
          {detail}
        </span>
      </span>
    </div>
  );
}

/** DockTool is one of the editor's tools: an icon, with its words when there is room, and a count when something waits in it. */
export function DockTool({
  active,
  count,
  icon: Icon,
  label,
  onClick,
  words,
}: {
  active?: boolean;
  count?: number;
  icon: LucideIcon;
  label: string;
  onClick: () => void;
  words?: string;
}) {
  return (
    <Tooltip content={label}>
      <button
        aria-label={label}
        aria-pressed={active}
        className={cn(
          "relative inline-flex h-control shrink-0 items-center gap-2 rounded-control px-2.5 text-meta font-medium text-mute transition-colors duration-80 hover:bg-fill hover:text-ink aria-pressed:bg-accent-wash aria-pressed:text-accent",
          focusRing,
        )}
        onClick={onClick}
        type="button"
      >
        <Icon aria-hidden="true" className="size-4.5" />
        {words ? (
          <span aria-hidden="true" className="max-xl:sr-only">
            {words}
          </span>
        ) : null}
        {count ? (
          <span
            aria-hidden="true"
            className="inline-flex min-w-5 items-center justify-center rounded-full bg-action px-1.5 text-label leading-5 font-semibold text-on-accent tabular-nums max-xl:absolute max-xl:-top-1 max-xl:-right-1"
          >
            {count}
          </span>
        ) : null}
      </button>
    </Tooltip>
  );
}

/** DockAction is a dock's action: violet when it is the main one, grey otherwise. */
export function DockAction({
  children,
  disabled,
  onClick,
  strong,
}: {
  children: ReactNode;
  disabled?: boolean;
  onClick: () => void;
  strong?: boolean;
}) {
  return (
    <Button
      disabled={disabled}
      onClick={onClick}
      variant={strong ? "primary" : "secondary"}
    >
      {children}
    </Button>
  );
}

/** Dock is a writing surface's control strip at the bottom of the screen: the save state, the tools and the actions. */
export function Dock({
  actions,
  detail,
  railOpen,
  state,
  tools,
  words,
}: {
  actions: ReactNode;
  detail: string;
  railOpen: boolean;
  state: DockState;
  tools: ReactNode;
  words: string;
}) {
  return (
    <div
      className={cn(
        "pointer-events-none fixed inset-x-0 bottom-0 z-30 flex justify-center px-3 pb-3 transition-[padding] duration-240 ease-wipe md:pb-6",
        railOpen && "max-lg:hidden lg:pr-[28rem]",
      )}
    >
      <div className="pointer-events-auto flex w-full max-w-[50rem] items-center gap-1.5 rounded-card bg-plane p-1.5 shadow-popover ring-1 ring-ink/8 md:gap-2 md:p-2">
        <div
          aria-live="polite"
          className="flex min-w-0 flex-1 items-center gap-2.5 px-2 md:px-3"
        >
          <StatusLight state={state} />
          <span className="min-w-0 leading-4">
            <span className="block truncate text-meta font-medium text-ink">
              {words}
            </span>
            <span className="block truncate text-label text-mute max-md:sr-only">
              {detail}
            </span>
          </span>
        </div>
        <div className="flex items-center gap-0.5">{tools}</div>
        <div className="flex shrink-0 items-center gap-1.5">{actions}</div>
      </div>
    </div>
  );
}
