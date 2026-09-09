"use client";

import { ChevronDown } from "lucide-react";
import { useState } from "react";
import type { RecordedVersion } from "@/lib/api/query";
import { versionAnchor, versionDate, versionTitle } from "@/lib/asset-updates";
import { cn } from "@/lib/cn";
import { useReadingMark } from "@/lib/use-reading-mark";

/** Every recorded version as a way into it, in the margin on a wide screen and a disclosure on a narrow one. */
export function VersionSpine({ versions }: { versions: RecordedVersion[] }) {
  const [open, setOpen] = useState(false);
  const here = useReadingMark(versions.map(versionAnchor));

  return (
    <nav aria-label="Recorded versions" className="group" data-open={open}>
      <p aria-hidden="true" className="hidden text-meta font-medium lg:block">
        Every version
      </p>
      <button
        aria-controls="version-spine"
        aria-expanded={open}
        className="flex min-h-11 w-full items-center justify-between gap-3 rounded-control bg-deep px-4 text-ui font-medium text-ink outline-offset-3 lg:hidden"
        onClick={() => setOpen(!open)}
        type="button"
      >
        Every version
        <ChevronDown
          aria-hidden="true"
          className="size-4 transition-transform duration-200 group-data-[open=true]:rotate-180 motion-reduce:transition-none"
        />
      </button>
      <ol
        className="mt-2 hidden list-none group-data-[open=true]:block lg:mt-3 lg:block"
        id="version-spine"
      >
        {versions.map((version) => {
          const anchor = versionAnchor(version);
          return (
            <li key={version.id}>
              <a
                aria-current={anchor === here ? "location" : undefined}
                className={cn(
                  "flex min-h-11 flex-col justify-center py-1 outline-offset-3",
                  "text-ui text-mute transition-colors hover:text-ink motion-reduce:transition-none",
                  "aria-[current=location]:font-medium aria-[current=location]:text-accent",
                )}
                href={`#${anchor}`}
                onClick={() => setOpen(false)}
              >
                {versionTitle(version)}
                <span className="text-meta text-mute">
                  {versionDate(version)}
                </span>
              </a>
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
