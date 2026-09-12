"use client";

import Link from "next/link";
import { useState } from "react";
import { cn } from "@/lib/cn";

export type ChipItem = {
  id: string;
  label: string;
  href?: string;
};

const CHIP =
  "inline-flex min-h-8 max-w-full items-center justify-center rounded-control bg-deep px-2.5 py-1 text-center font-prose text-label text-mute [overflow-wrap:anywhere]";

export function ChipSet({
  items,
  limit,
  className,
}: {
  items: readonly ChipItem[];
  limit?: number;
  className?: string;
}) {
  const [showingAll, setShowingAll] = useState(false);
  const held = limit === undefined ? 0 : Math.max(items.length - limit, 0);
  const shown = held > 0 && !showingAll ? items.slice(0, limit) : items;

  return (
    <div className={className}>
      <ul className="flex list-none flex-wrap items-start gap-1.5">
        {shown.map((item) => (
          <li key={item.id}>
            {item.href ? (
              <Link
                className={cn(
                  CHIP,
                  "transition-colors duration-200 hover:bg-accent-wash hover:text-ink motion-reduce:transition-none",
                )}
                href={item.href}
              >
                {item.label}
              </Link>
            ) : (
              <span className={CHIP}>{item.label}</span>
            )}
          </li>
        ))}
        {held > 0 ? (
          <li>
            <button
              aria-expanded={showingAll}
              className={cn(
                CHIP,
                "cursor-pointer bg-transparent font-medium whitespace-nowrap text-ink hover:bg-deep",
              )}
              onClick={() => setShowingAll((current) => !current)}
              type="button"
            >
              {showingAll ? "Fewer" : `${held} more`}
            </button>
          </li>
        ) : null}
      </ul>
    </div>
  );
}
