"use client";

import Link from "next/link";
import type { MouseEvent, ReactNode } from "react";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import { useChoicePlates } from "@/components/ui/radio-group";
import { WeightLabel } from "@/components/ui/weight-label";
import { cn, focusRing } from "@/lib/cn";

type SubtleTab<T extends string> = {
  value: T;
  label: ReactNode;
  count?: ReactNode;
  attention?: boolean;
  href?: string;
};

const TAB = `relative z-10 flex h-control shrink-0 cursor-pointer items-center gap-2 rounded-control px-3 font-ui text-ui whitespace-nowrap transition-colors duration-80 ${focusRing}`;

/** SubtleTabs is Fluid Functionalism's subtle tabs for moving between sections of a page: a violet plate slides to the section shown and a hover plate follows the pointer. */
export function SubtleTabs<T extends string>({
  tabs,
  chosen,
  onChoose,
  className,
}: {
  tabs: readonly SubtleTab<T>[];
  chosen: T;
  onChoose?: (value: T) => void;
  className?: string;
}) {
  const { containerRef, hover, itemRefs, chosenPlate } = useChoicePlates(
    tabs,
    chosen,
    "x",
  );
  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: the pointer only moves the hover plate; each tab is its own link or button
    <div
      className={cn("relative flex w-max gap-1 p-1 select-none", className)}
      onMouseEnter={hover.handlers.onMouseEnter}
      onMouseLeave={hover.handlers.onMouseLeave}
      onMouseMove={hover.handlers.onMouseMove}
      ref={containerRef}
    >
      {chosenPlate("rounded-control bg-accent-wash")}
      <FluidHoverHighlight className="rounded-control" hover={hover} />
      {tabs.map((tab, index) => {
        const here = tab.value === chosen;
        const shared = {
          "aria-current": here ? ("page" as const) : undefined,
          className: cn(TAB, here ? "text-accent" : "text-mute hover:text-ink"),
          "data-fluid-hover-index": index,
          onFocus: () => hover.setActiveIndex(index),
          ref: itemRefs[index],
        };
        const inside = (
          <>
            <WeightLabel chosen={here}>{tab.label}</WeightLabel>
            {tab.count === undefined ? null : (
              <span
                className={cn(
                  "font-prose text-meta tabular-nums",
                  tab.attention && !here
                    ? "rounded-control bg-stop-wash px-1.5 text-stop"
                    : here
                      ? "text-accent"
                      : "text-mute",
                )}
              >
                {tab.count}
              </span>
            )}
          </>
        );
        if (tab.href) {
          return (
            <Link
              {...shared}
              href={tab.href}
              key={tab.value}
              onClick={(event: MouseEvent) => {
                if (!onChoose) return;
                if (event.metaKey || event.ctrlKey || event.shiftKey) return;
                event.preventDefault();
                onChoose(tab.value);
              }}
            >
              {inside}
            </Link>
          );
        }
        return (
          <button
            {...shared}
            key={tab.value}
            onClick={() => onChoose?.(tab.value)}
            type="button"
          >
            {inside}
          </button>
        );
      })}
    </div>
  );
}
