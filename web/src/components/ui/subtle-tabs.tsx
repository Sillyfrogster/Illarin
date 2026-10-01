"use client";

import Link from "next/link";
import type { MouseEvent, ReactNode } from "react";
import { cn, focusRing } from "@/lib/cn";

type SubtleTab<T extends string> = {
  value: T;
  label: ReactNode;
  count?: ReactNode;
  attention?: boolean;
  href?: string;
};

const TAB = `flex h-control shrink-0 cursor-pointer items-center gap-2 rounded-control px-3 font-ui text-ui font-medium whitespace-nowrap transition-colors duration-80 ${focusRing}`;

/** SubtleTabs moves between sections of a page: the section shown is violet on the violet wash, the rest grey until hovered. */
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
  return (
    <div className={cn("flex w-max gap-1 p-1 select-none", className)}>
      {tabs.map((tab) => {
        const here = tab.value === chosen;
        const shared = {
          "aria-current": here ? ("page" as const) : undefined,
          className: cn(
            TAB,
            here
              ? "bg-accent-wash text-accent hover:text-accent"
              : "text-mute hover:bg-fill hover:text-ink",
          ),
        };
        const inside = (
          <>
            {tab.label}
            {tab.count === undefined ? null : (
              <span
                className={cn(
                  "font-normal text-meta tabular-nums",
                  tab.attention && !here
                    ? "rounded-chip bg-stop-wash px-1.5 text-stop"
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
