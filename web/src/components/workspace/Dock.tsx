"use client";

import { motion, useReducedMotion } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

export type DockState =
  | "failed"
  | "unsaved"
  | "saving"
  | "private"
  | "published";

export const TOOL =
  "relative inline-flex size-11 shrink-0 items-center justify-center rounded-control text-field opacity-70 outline-offset-3 hover:bg-field/15 hover:opacity-100 focus-visible:opacity-100 aria-pressed:bg-field/15 aria-pressed:opacity-100";

const QUIET_ACTION =
  "inline-flex min-h-11 items-center rounded-control px-3.5 text-ui font-medium text-field opacity-75 outline-offset-3 hover:bg-field/15 hover:opacity-100 disabled:opacity-35";

const STRONG_ACTION =
  "inline-flex min-h-11 shrink-0 items-center rounded-control bg-field px-5 text-ui font-medium text-ink outline-offset-3 hover:opacity-90 disabled:opacity-35";

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
          state === "private" && "shadow-[inset_0_0_0_2px_var(--v-action)]",
          state === "published" && "bg-field/60",
        )}
      />
    </span>
  );
}

export const WORDED_TOOL = "w-auto gap-2 px-3 text-meta font-medium";

export function DockTool({
  active,
  count,
  icon: Icon,
  label,
  onClick,
  worded,
}: {
  active?: boolean;
  count?: number;
  icon: LucideIcon;
  label: string;
  onClick: () => void;
  worded?: boolean;
}) {
  return (
    <button
      aria-label={worded ? undefined : label}
      aria-pressed={active}
      className={cn(TOOL, worded && WORDED_TOOL)}
      onClick={onClick}
      title={worded ? undefined : label}
      type="button"
    >
      <Icon aria-hidden="true" size={18} />
      {worded ? label : null}
      {count ? (
        <span
          aria-hidden="true"
          className="absolute -top-0.5 -right-0.5 inline-flex min-w-4 items-center justify-center rounded-full bg-accent px-1 font-ui text-[0.62rem] leading-4 font-semibold text-on-accent tabular-nums"
        >
          {count}
        </span>
      ) : null}
    </button>
  );
}

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

export function Dock({
  actions,
  detail,
  layoutId,
  railOpen,
  state,
  tools,
  words,
}: {
  actions: ReactNode;
  detail: string;
  layoutId?: string;
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
        initial={
          layoutId && !reduced
            ? false
            : reduced
              ? { opacity: 0 }
              : { opacity: 0, y: 96 }
        }
        layoutId={reduced ? undefined : layoutId}
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

        <div className="flex flex-1 basis-auto flex-wrap items-center gap-0.5 sm:justify-center">
          {tools}
        </div>

        <div className="flex shrink-0 items-center gap-1">{actions}</div>
      </motion.div>
    </div>
  );
}
