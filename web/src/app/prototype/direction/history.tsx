"use client";

import { useState } from "react";
import type { Release } from "./assets";
import { Counter } from "./motion";
import { Action, type Direction, Label, Rule } from "./ui";

function Change({ value, sign }: { value: number; sign: "+" | "−" }) {
  return (
    <span
      className="v-tabular vd:text-meta vd:font-semibold"
      style={{
        color: sign === "+" ? "var(--v-accent)" : "var(--v-mute)",
        opacity: value ? 1 : 0.35,
      }}
    >
      {sign}
      <Counter value={value} />
    </span>
  );
}

/** The update record, opened one version at a time so the whole history never lands at once */
export function History({
  releases,
  direction,
  number,
}: {
  releases: Release[];
  direction: Direction;
  number?: number;
}) {
  const [openId, setOpenId] = useState(releases[0]?.version);

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
          Update record
        </h2>
        <p className="vd:text-meta vd:text-mute">
          {releases.length} recorded{" "}
          {releases.length === 1 ? "version" : "versions"}
        </p>
      </div>

      <ul className="vd:mt-group">
        {releases.map((release, index) => {
          const open = release.version === openId;
          const previous = releases[index + 1];
          return (
            <li key={release.version}>
              <Rule />
              <button
                type="button"
                aria-expanded={open}
                onClick={() => setOpenId(open ? "" : release.version)}
                className="vd:flex vd:w-full vd:flex-wrap vd:items-baseline vd:gap-x-6 vd:gap-y-2 vd:py-5 vd:text-left"
              >
                <span
                  className="v-tabular vd:font-display vd:text-[1.75rem] vd:leading-none vd:font-medium"
                  style={{ color: open ? "var(--v-accent)" : undefined }}
                >
                  {release.version}
                </span>
                <span className="vd:min-w-0 vd:flex-1 vd:text-ui vd:text-mute">
                  {release.summary}
                </span>
                <span className="vd:flex vd:shrink-0 vd:items-baseline vd:gap-3">
                  <Change value={release.additions} sign="+" />
                  <Change value={release.removals} sign="−" />
                </span>
                <span className="v-tabular vd:w-40 vd:shrink-0 vd:text-right vd:text-meta vd:text-faint">
                  {release.date}
                </span>
              </button>
              {open && (
                <div className="vd:grid vd:gap-6 vd:pb-8 vd:sm:grid-cols-[10rem_1fr]">
                  <Label>What changed</Label>
                  <div className="vd:min-w-0">
                    <ul className="vd:grid vd:gap-2.5">
                      {release.changes.map((change) => (
                        <li
                          key={change}
                          className="vd:flex vd:gap-4 vd:text-ui vd:leading-7"
                        >
                          <span
                            aria-hidden="true"
                            className="vd:mt-[0.8em] vd:h-px vd:w-5 vd:shrink-0 vd:bg-accent"
                          />
                          {change}
                        </li>
                      ))}
                    </ul>
                    <div className="vd:mt-6 vd:flex vd:flex-wrap vd:gap-3">
                      {previous && (
                        <Action variant="secondary">
                          Compare with {previous.version}
                        </Action>
                      )}
                      <Action variant="quiet">
                        Download {release.version}
                      </Action>
                    </div>
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
