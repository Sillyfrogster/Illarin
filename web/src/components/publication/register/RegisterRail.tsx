"use client";

import { useState } from "react";
import { TravellingHighlight } from "@/components/ui/travelling-highlight";
import { cn } from "@/lib/cn";
import {
  REGISTERS,
  type Register,
  type RegisterStanding,
  registerName,
} from "@/lib/publication-register";

/** Chooses which register the page shows, with a count on each. */
export function RegisterRail({
  chosen,
  onChoose,
  standings,
}: {
  chosen: Register;
  onChoose: (register: Register) => void;
  standings: Record<Register, RegisterStanding>;
}) {
  const [lit, setLit] = useState<string>(chosen);

  return (
    <nav
      aria-label="What the publication keeps"
      className="-mx-[var(--gutter)] min-w-0"
    >
      <div className="overflow-x-auto px-[var(--gutter)] pb-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <TravellingHighlight
          chosen={chosen}
          className="flex w-max gap-1"
          onLit={setLit}
        >
          {REGISTERS.map((register) => {
            const standing = standings[register];
            const on = lit === register;
            return (
              <button
                aria-current={chosen === register ? "true" : undefined}
                className={cn(
                  "flex min-h-11 items-center gap-2 rounded-control px-4 font-ui text-ui font-medium whitespace-nowrap outline-offset-2 transition-colors duration-200 motion-reduce:transition-none",
                  on ? "text-on-accent" : "text-mute",
                )}
                data-cell={register}
                key={register}
                onClick={() => onChoose(register)}
                type="button"
              >
                {registerName(register)}
                {standing.count === null ? null : (
                  <span
                    className={cn(
                      "font-prose text-meta tabular-nums",
                      on
                        ? "opacity-70"
                        : standing.attention
                          ? "rounded-control bg-stop-wash px-1.5 text-stop opacity-100"
                          : "opacity-60",
                    )}
                  >
                    {standing.count}
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
