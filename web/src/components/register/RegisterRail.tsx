"use client";

import { useState } from "react";
import { TravellingHighlight } from "@/components/ui/travelling-highlight";
import { cn } from "@/lib/cn";

export type RegisterCell<Id extends string> = {
  attention?: boolean;
  count: number | null;
  id: Id;
  name: string;
};

export function RegisterRail<Id extends string>({
  cells,
  chosen,
  label,
  onChoose,
}: {
  cells: RegisterCell<Id>[];
  chosen: Id;
  label: string;
  onChoose: (id: Id) => void;
}) {
  const [lit, setLit] = useState<string>(chosen);

  return (
    <nav aria-label={label} className="-mx-[var(--gutter)] min-w-0">
      <div className="overflow-x-auto px-[var(--gutter)] pb-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <TravellingHighlight
          chosen={chosen}
          className="flex w-max gap-1"
          onLit={setLit}
        >
          {cells.map((cell) => {
            const on = lit === cell.id;
            return (
              <button
                aria-current={chosen === cell.id ? "true" : undefined}
                className={cn(
                  "flex min-h-11 items-center gap-2 rounded-control px-4 font-ui text-ui font-medium whitespace-nowrap outline-offset-2 transition-colors duration-200 motion-reduce:transition-none",
                  on ? "text-on-accent" : "text-mute",
                )}
                data-cell={cell.id}
                key={cell.id}
                onClick={() => onChoose(cell.id)}
                type="button"
              >
                {cell.name}
                {cell.count === null ? null : (
                  <span
                    className={cn(
                      "font-prose text-meta tabular-nums",
                      on
                        ? "opacity-70"
                        : cell.attention
                          ? "rounded-control bg-stop-wash px-1.5 text-stop opacity-100"
                          : "opacity-60",
                    )}
                  >
                    {cell.count}
                  </span>
                )}
              </button>
            );
          })}
        </TravellingHighlight>
      </div>
    </nav>
  );
}
