"use client";

import { useState } from "react";
import type { Format } from "./assets";
import { cn, type Direction, Rule } from "./ui";

const VERDICT: Record<Format["verdict"], string> = {
  carried: "Carries everything on this page",
  reduced: "Leaves some of this page behind",
  blocked: "Cannot carry this asset",
};

/** A reader chooses a download knowing what it drops, which is the whole point of showing this */
export function Formats({
  formats,
  direction,
  number,
}: {
  formats: Format[];
  direction: Direction;
  number?: number;
}) {
  const [openId, setOpenId] = useState(
    formats.find((format) => format.recommended)?.label ?? formats[0]?.label,
  );

  return (
    <div>
      <div className="vd:flex vd:flex-wrap vd:items-baseline vd:justify-between vd:gap-4">
        <h2 className="vd:flex vd:items-baseline vd:gap-4 vd:font-display vd:text-title vd:font-medium">
          {direction === "ledger" && number && (
            <span
              className="v-tabular vd:font-normal"
              style={{ color: "var(--v-accent)" }}
            >
              {String(number).padStart(2, "0")}
            </span>
          )}
          Downloads
        </h2>
        <p className="vd:text-meta vd:text-mute">
          {formats.length} {formats.length === 1 ? "format" : "formats"}
        </p>
      </div>

      <ul className="vd:mt-group">
        {formats.map((format) => {
          const open = format.label === openId;
          return (
            <li key={format.label}>
              <Rule />
              <button
                type="button"
                aria-expanded={open}
                onClick={() => setOpenId(open ? "" : format.label)}
                className="vd:grid vd:w-full vd:gap-x-8 vd:gap-y-1 vd:py-5 vd:text-left vd:sm:grid-cols-[16rem_1fr]"
              >
                <span className="vd:min-w-0">
                  <span
                    className="vd:block vd:font-display vd:text-[1.5rem] vd:leading-tight vd:font-medium"
                    style={{ color: open ? "var(--v-accent)" : undefined }}
                  >
                    {format.label}
                  </span>
                  {format.recommended && (
                    <span className="vd:mt-1 vd:block vd:text-label vd:font-bold vd:uppercase vd:text-mute">
                      Recommended
                    </span>
                  )}
                </span>
                <span className="vd:flex vd:min-w-0 vd:flex-wrap vd:items-baseline vd:justify-between vd:gap-x-6 vd:gap-y-1">
                  <span className="vd:text-ui vd:text-mute">{format.note}</span>
                  <span
                    className={cn(
                      "vd:shrink-0 vd:text-meta",
                      format.verdict === "carried"
                        ? "vd:text-mute"
                        : "vd:text-warn",
                    )}
                  >
                    {VERDICT[format.verdict]}
                  </span>
                </span>
              </button>
              {open && (
                <div className="vd:grid vd:gap-x-8 vd:gap-y-3 vd:pb-8 vd:sm:grid-cols-[16rem_1fr]">
                  <p className="vd:text-label vd:font-bold vd:uppercase vd:text-mute">
                    {format.drops ? "Not carried" : "Carried"}
                  </p>
                  <div className="vd:min-w-0">
                    {format.drops ? (
                      <ul className="vd:grid vd:gap-2.5">
                        {format.drops.map((drop) => (
                          <li
                            key={drop}
                            className="vd:flex vd:gap-4 vd:text-ui vd:leading-7"
                          >
                            <span
                              aria-hidden="true"
                              className="vd:mt-[0.8em] vd:h-px vd:w-5 vd:shrink-0 vd:bg-warn"
                            />
                            {drop}
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="vd:max-w-[56ch] vd:text-ui vd:leading-7 vd:text-mute">
                        Every element on this page survives this download,
                        including the parts Illarin could not read when the file
                        arrived.
                      </p>
                    )}
                  </div>
                </div>
              )}
            </li>
          );
        })}
      </ul>
      <Rule />
    </div>
  );
}
