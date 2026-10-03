"use client";

import { Check, Minus } from "lucide-react";
import { useState } from "react";
import type { BrowseType } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { TYPE_PLURALS } from "@/lib/work-types";
import { HEADING } from "./heading";

/** APPS is what each app reads, copied from the format registry in api/internal/format/capability.go. */
const APPS: { name: string; reads: BrowseType[] }[] = [
  {
    name: "SillyTavern",
    reads: ["character", "lorebook", "preset", "theme", "extension"],
  },
  { name: "RisuAI", reads: ["character", "lorebook"] },
  {
    name: "Lumiverse",
    reads: ["character", "lorebook", "preset", "theme", "pack", "extension"],
  },
];
const TYPES: BrowseType[] = [
  "character",
  "lorebook",
  "preset",
  "theme",
  "pack",
  "extension",
];

/** AppReads lets the reader pick their app and shows which types Illarin hands it, with the download button that results. */
export function AppReads() {
  const [at, setAt] = useState(0);
  const app = APPS[at];

  return (
    <div className="grid items-center gap-section lg:grid-cols-2">
      <div className="flex flex-col gap-6">
        <h2 className={cn(HEADING, "max-w-[16ch]")} id="home-apps">
          Pick your app, Illarin does the rest
        </h2>
        <p className="max-w-[44ch] text-lede text-mute">
          Every work page offers the version your app reads. Choose yours to see
          what it opens.
        </p>
        <fieldset className="m-0 flex w-fit min-w-0 flex-wrap gap-1 rounded-full border-0 bg-inset p-1">
          <legend className="sr-only">Your app</legend>
          {APPS.map((one, index) => (
            <button
              aria-pressed={index === at}
              className={cn(
                "h-9 rounded-full px-4 text-ui font-medium transition-colors duration-200",
                index === at
                  ? "bg-plane text-ink shadow-card"
                  : "text-mute hover:text-ink",
              )}
              key={one.name}
              onClick={() => setAt(index)}
              type="button"
            >
              {one.name}
            </button>
          ))}
        </fieldset>
      </div>
      <div className="rounded-card bg-plane p-2 shadow-card ring-1 ring-ink/8">
        <ul className="m-0 grid list-none grid-cols-1 p-0 sm:grid-cols-2">
          {TYPES.map((type) => {
            const reads = app.reads.includes(type);
            return (
              <li
                className="flex items-center justify-between gap-3 rounded-art px-4 py-3"
                key={type}
              >
                <span
                  className={cn(
                    "text-ui font-medium transition-colors duration-300",
                    reads ? "text-ink" : "text-mute",
                  )}
                >
                  {TYPE_PLURALS[type]}
                </span>
                <span
                  className={cn(
                    "grid size-6 place-items-center rounded-full transition-[background-color,color,scale] duration-300 ease-(--ease-wipe)",
                    reads
                      ? "scale-100 bg-action text-on-accent"
                      : "scale-90 bg-inset text-mute",
                  )}
                >
                  {reads ? (
                    <Check aria-label="Opens" className="size-3.5" />
                  ) : (
                    <Minus aria-label="Doesn't open" className="size-3.5" />
                  )}
                </span>
              </li>
            );
          })}
        </ul>
        <div className="mt-2 flex flex-wrap items-center justify-between gap-3 rounded-art bg-inset px-4 py-3">
          <span className="text-meta text-mute">On every work page</span>
          <span
            aria-hidden="true"
            className="inline-flex h-9 items-center rounded-full bg-action px-4 text-ui font-medium text-on-accent"
          >
            Download for {app.name}
          </span>
        </div>
      </div>
    </div>
  );
}
