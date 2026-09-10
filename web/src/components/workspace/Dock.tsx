"use client";

import { motion, useReducedMotion } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

/** How the work stands with the server, which is what the light says without words. */
export type DockState =
  | "failed"
  | "unsaved"
  | "saving"
  | "private"
  | "published";

const TOOL =
  "relative inline-flex size-11 shrink-0 items-center justify-center rounded-control text-field opacity-70 outline-offset-3 hover:bg-field/15 hover:opacity-100 focus-visible:opacity-100 aria-pressed:bg-field/15 aria-pressed:opacity-100";

const QUIET_ACTION =
  "inline-flex min-h-11 items-center rounded-control px-3.5 text-ui font-medium text-field opacity-75 outline-offset-3 hover:bg-field/15 hover:opacity-100 disabled:opacity-35";

const STRONG_ACTION =
  "inline-flex min-h-11 shrink-0 items-center rounded-control bg-field px-5 text-ui font-medium text-ink outline-offset-3 hover:opacity-90 disabled:opacity-35";

/** Private work is a ring and published work a filled dot, so the two never read alike. */
export function StatusLight({ state }: { state: DockState }) {
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

/** One of the ways the dock opens something beside the page. */
export function DockTool({
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
      className={TOOL}
      onClick={onClick}
      title={label}
      type="button"
    >
      <Icon aria-hidden="true" size={18} />
    </button>
  );
}

/** A dock action that gives way to the one beside it. */
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
    <button
      className={strong ? STRONG_ACTION : QUIET_ACTION}
      disabled={disabled}
      onClick={onClick}
      type="button"
    >
      {children}
    </button>
  );
}

/**
 * One bar at the foot of a workspace carrying how the work stands, the ways to
 * open what sits beside the page, and the action that reaches readers. It moves
 * clear of the rail on a wide screen and steps out from under it on a narrow one.
 */
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
  const reduced = useReducedMotion();

  return (
    <div
      className={cn(
        "pointer-events-none fixed inset-x-0 bottom-0 z-30 flex justify-center px-3 pb-4 transition-[padding] duration-500 ease-wipe motion-reduce:transition-none md:pb-7",
        railOpen && "max-lg:hidden lg:pr-[28rem]",
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
          <StatusLight state={state} />
          <span className="min-w-0 leading-4">
            <span className="block truncate text-meta font-medium">
              {words}
            </span>
            <span className="block truncate text-label opacity-65">
              {detail}
            </span>
          </span>
        </div>

        <div className="flex min-w-0 flex-1 items-center gap-0.5 sm:justify-center">
          {tools}
        </div>

        <div className="flex shrink-0 items-center gap-1">{actions}</div>
      </motion.div>
    </div>
  );
}
