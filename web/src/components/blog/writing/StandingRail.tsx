"use client";

import { useState } from "react";
import { TravellingHighlight } from "@/components/ui/travelling-highlight";
import { cn } from "@/lib/cn";
import { STANDINGS, type Standing, standingName } from "@/lib/post-standing";

export function StandingRail({
  chosen,
  counts,
  onChoose,
}: {
  chosen: Standing;
  counts: Record<Standing, number>;
  onChoose: (standing: Standing) => void;
}) {
  const [lit, setLit] = useState<string>(chosen);

  return (
    <nav aria-label="Which posts" className="-mx-[var(--gutter)] min-w-0">
      <div className="overflow-x-auto px-[var(--gutter)] pb-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <TravellingHighlight
          chosen={chosen}
          className="flex w-max gap-1"
          onLit={setLit}
        >
          {STANDINGS.map((standing) => (
            <button
              aria-current={chosen === standing ? "true" : undefined}
              className={cn(
                "flex min-h-11 items-center gap-2 rounded-control px-4 font-ui text-ui font-medium whitespace-nowrap outline-offset-2 transition-colors duration-200 motion-reduce:transition-none",
                lit === standing ? "text-on-accent" : "text-mute",
              )}
              data-cell={standing}
              key={standing}
              onClick={() => onChoose(standing)}
              type="button"
            >
              {standingName(standing)}
              <span
                className={cn(
                  "font-prose text-meta tabular-nums",
                  lit === standing ? "opacity-70" : "opacity-60",
                )}
              >
                {counts[standing]}
              </span>
            </button>
          ))}
        </TravellingHighlight>
      </div>
    </nav>
  );
}
