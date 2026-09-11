"use client";

import { motion, useReducedMotion } from "framer-motion";
import { Maximize2 } from "lucide-react";
import {
  createContext,
  type ReactNode,
  useContext,
  useId,
  useState,
} from "react";
import { cn } from "@/lib/cn";

const GLIDE = { type: "spring", stiffness: 260, damping: 30 } as const;

const RunContext = createContext<{
  group: string;
  lit: string | null;
  light: (key: string | null) => void;
} | null>(null);

/** RunOpenContext lets each row open itself somewhere roomier than the run. */
export const RunOpenContext = createContext<((key: string) => void) | null>(
  null,
);

/** Run is a list of items under one plate that glides to the one you are reading. */
export function Run({
  as: Tag = "ul",
  children,
  className,
}: {
  as?: "ul" | "ol" | "dl";
  children: ReactNode;
  className?: string;
}) {
  const [lit, setLit] = useState<string | null>(null);
  const group = useId();

  return (
    <RunContext.Provider value={{ group, light: setLit, lit }}>
      <Tag
        className={cn(
          "flex list-none flex-col overflow-hidden rounded-plate bg-inset",
          className,
        )}
        onBlur={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget)) setLit(null);
        }}
        onMouseLeave={() => setLit(null)}
      >
        {children}
      </Tag>
    </RunContext.Provider>
  );
}

export function RunHeading({
  children,
  count,
}: {
  children: ReactNode;
  count?: string;
}) {
  return (
    <li className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 bg-deep/70 px-4 py-2.5">
      <span className="font-ui text-meta font-semibold tracking-[0.02em] text-ink">
        {children}
      </span>
      {count ? (
        <span className="font-ui text-label text-mute tabular-nums">
          {count}
        </span>
      ) : null}
    </li>
  );
}

export function RunItem({
  as: Tag = "li",
  children,
  className,
  itemKey,
}: {
  as?: "li" | "div";
  children: ReactNode;
  className?: string;
  itemKey: string;
}) {
  const run = useContext(RunContext);
  const open = useContext(RunOpenContext);
  const still = useReducedMotion();

  return (
    <Tag
      className={cn(
        "group/row relative isolate flex min-w-0 flex-col gap-1.5 px-4 py-3.5 not-first:border-rule/45 not-first:border-t",
        open && "pr-14",
        className,
      )}
      onFocus={() => run?.light(itemKey)}
      onMouseEnter={() => run?.light(itemKey)}
    >
      {run?.lit === itemKey ? (
        <motion.span
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 -z-1 bg-accent-wash inset-ring-1 inset-ring-accent/25"
          layoutId={`run-plate-${run.group}`}
          transition={still ? { duration: 0 } : GLIDE}
        />
      ) : null}
      {children}
      {open ? (
        <button
          aria-label="Open in the browser"
          className="absolute top-2.5 right-2.5 inline-flex size-9 items-center justify-center rounded-control text-mute opacity-0 outline-offset-2 transition-[opacity,background-color,color] duration-200 group-hover/row:opacity-100 hover:bg-accent-wash hover:text-accent focus-visible:opacity-100 motion-reduce:transition-none"
          onClick={() => open(itemKey)}
          type="button"
        >
          <Maximize2 aria-hidden="true" className="size-4" />
        </button>
      ) : null}
    </Tag>
  );
}
