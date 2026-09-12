"use client";

import { useReducedMotion } from "framer-motion";
import { ChevronDown, Maximize2 } from "lucide-react";
import { type ReactNode, useLayoutEffect, useRef } from "react";
import { cn } from "@/lib/cn";

const CONTROL =
  "group/unfold relative isolate mt-1 inline-flex min-h-11 items-center gap-2 self-start overflow-hidden rounded-control bg-deep px-4 font-ui text-meta font-medium text-ink outline-offset-3";

/** The longest a wipe may run, however much content it uncovers. */
const SETTLE_CEILING_MS = 620;

const SETTLE_PER_PX = 0.34;

/** Unfold opens an excerpt where it sits rather than in a second copy of it. */
export function Unfold({
  children,
  id,
  isCut,
  more,
  onToggle,
  open,
  panelId,
}: {
  children: ReactNode;
  id: string;
  isCut: boolean;
  more: string;
  onToggle: () => void;
  open: boolean;
  panelId: string;
}) {
  const shell = useRef<HTMLDivElement>(null);
  const region = useRef<HTMLDivElement>(null);
  const measured = useRef<number | null>(null);
  const still = useReducedMotion();

  // biome-ignore lint/correctness/useExhaustiveDependencies: Opening and closing is what asks for the new measurement.
  useLayoutEffect(() => {
    const box = shell.current;
    const inner = region.current;
    if (!box || !inner) return;

    const from = measured.current;
    const to = inner.getBoundingClientRect().height;
    measured.current = to;
    if (from === null || Math.abs(to - from) < 1 || still) return;

    const settle = Math.min(
      SETTLE_CEILING_MS,
      180 + Math.abs(to - from) * SETTLE_PER_PX,
    );
    const rest = () => {
      box.style.transition = "";
      box.style.height = "";
      box.style.overflow = "";
    };

    box.style.overflow = "hidden";
    box.style.transition = "";
    box.style.height = `${from}px`;
    void box.getBoundingClientRect().height;
    box.style.transition = `height ${settle}ms var(--ease-wipe)`;
    box.style.height = `${to}px`;
    box.addEventListener("transitionend", rest, { once: true });

    return () => {
      box.removeEventListener("transitionend", rest);
      rest();
    };
  }, [open, still]);

  return (
    <>
      <div className="min-w-0" ref={shell}>
        <div ref={region}>{children}</div>
      </div>
      {isCut || open ? (
        <button
          aria-controls={panelId}
          aria-expanded={open}
          className={CONTROL}
          data-read-more
          id={id}
          onClick={onToggle}
          type="button"
        >
          <span
            aria-hidden="true"
            className="absolute inset-0 -z-1 origin-left scale-x-0 bg-accent-wash transition-transform duration-300 ease-[var(--ease-wipe)] group-hover/unfold:scale-x-100 group-focus-visible/unfold:scale-x-100 motion-reduce:transition-none"
          />
          <ChevronDown
            aria-hidden="true"
            className={cn(
              "size-4 text-accent transition-transform duration-300 ease-[var(--ease-wipe)] motion-reduce:transition-none",
              open ? "rotate-180" : "rotate-0",
            )}
          />
          {open ? "Show less" : more}
        </button>
      ) : null}
    </>
  );
}

/** Browse shows a run's opening rows and hands the rest to a roomier surface. */
export function Browse({
  children,
  label,
  onOpen,
}: {
  children: ReactNode;
  label: string;
  onOpen: () => void;
}) {
  return (
    <>
      <div className="min-w-0">{children}</div>
      <button className={CONTROL} onClick={onOpen} type="button">
        <span
          aria-hidden="true"
          className="absolute inset-0 -z-1 origin-left scale-x-0 bg-accent-wash transition-transform duration-300 ease-[var(--ease-wipe)] group-hover/unfold:scale-x-100 group-focus-visible/unfold:scale-x-100 motion-reduce:transition-none"
        />
        <Maximize2 aria-hidden="true" className="size-4 text-accent" />
        {label}
      </button>
    </>
  );
}
