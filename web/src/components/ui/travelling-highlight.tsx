"use client";

import { type ReactNode, useCallback, useEffect, useRef } from "react";
import { cn } from "@/lib/cn";

const TRAVEL = 320;

const SHEEN =
  "radial-gradient(120% 120% at 50% 0%, rgb(255 255 255 / 0.26), rgb(255 255 255 / 0) 54%), linear-gradient(180deg, rgb(255 255 255 / 0) 55%, rgb(0 0 0 / 0.18))";

export function TravellingHighlight({
  chosen,
  children,
  className,
  onLit,
  plateClassName,
}: {
  chosen: string;
  children: ReactNode;
  className?: string;
  onLit?: (key: string) => void;
  plateClassName?: string;
}) {
  const row = useRef<HTMLDivElement>(null);
  const plate = useRef<HTMLSpanElement>(null);
  const settled = useRef(false);

  const moveTo = useCallback(
    (key: string, travel: boolean) => {
      const here = row.current;
      const mark = plate.current;
      const cell = here?.querySelector<HTMLElement>(
        `[data-cell="${CSS.escape(key)}"]`,
      );
      if (!here || !mark || !cell) return;

      const seat = cell.getBoundingClientRect();
      const frame = here.getBoundingClientRect();
      mark.style.transitionDuration = travel ? `${TRAVEL}ms` : "0s";
      mark.style.transform = `translate(${seat.left - frame.left}px, ${seat.top - frame.top}px)`;
      mark.style.width = `${seat.width}px`;
      mark.style.height = `${seat.height}px`;
      mark.style.opacity = "1";
      onLit?.(key);
    },
    [onLit],
  );

  useEffect(() => {
    moveTo(chosen, settled.current);
    settled.current = true;
  }, [chosen, moveTo]);

  useEffect(() => {
    const here = row.current;
    if (!here) return;
    const replace = () => moveTo(chosen, false);
    const watch = new ResizeObserver(replace);
    watch.observe(here);
    for (const cell of here.querySelectorAll("[data-cell]"))
      watch.observe(cell);
    window.addEventListener("resize", replace);
    return () => {
      watch.disconnect();
      window.removeEventListener("resize", replace);
    };
  }, [chosen, moveTo]);

  useEffect(() => {
    const here = row.current;
    if (!here) return;
    const follow = (event: Event) => {
      const cell = (event.target as HTMLElement | null)?.closest<HTMLElement>(
        "[data-cell]",
      );
      if (cell?.dataset.cell) moveTo(cell.dataset.cell, true);
    };
    const settle = () => moveTo(chosen, true);
    here.addEventListener("pointermove", follow);
    here.addEventListener("pointerleave", settle);
    here.addEventListener("focusin", follow);
    here.addEventListener("focusout", settle);
    return () => {
      here.removeEventListener("pointermove", follow);
      here.removeEventListener("pointerleave", settle);
      here.removeEventListener("focusin", follow);
      here.removeEventListener("focusout", settle);
    };
  }, [chosen, moveTo]);

  return (
    <div className={cn("relative isolate", className)} ref={row}>
      <span
        aria-hidden="true"
        className={cn(
          "pointer-events-none absolute top-0 left-0 -z-1 rounded-control bg-action opacity-0 motion-reduce:transition-none",
          plateClassName,
        )}
        ref={plate}
        style={{
          backgroundImage: SHEEN,
          transitionProperty: "transform, width, height, opacity",
          transitionTimingFunction: "var(--spring-smooth)",
        }}
      />
      {children}
    </div>
  );
}
