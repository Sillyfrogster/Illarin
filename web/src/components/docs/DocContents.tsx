"use client";

import { ChevronDown } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/cn";
import type { DocContentsEntry } from "@/lib/docs/read-doc";
import { useReadingMark } from "@/lib/use-reading-mark";

export function DocContents({ entries }: { entries: DocContentsEntry[] }) {
  const [open, setOpen] = useState(false);
  const here = useReadingMark(entries.map((entry) => entry.anchor));

  return (
    <nav aria-label="On this page" className="group" data-open={open}>
      <p
        aria-hidden="true"
        className="hidden font-ui text-meta font-semibold text-ink xl:block"
      >
        On this page
      </p>
      <button
        aria-controls="doc-contents"
        aria-expanded={open}
        className="flex min-h-11 w-full items-center justify-between gap-3 rounded-control bg-deep px-4 font-ui text-ui font-medium text-ink outline-offset-3 xl:hidden"
        onClick={() => setOpen(!open)}
        type="button"
      >
        On this page
        <ChevronDown
          aria-hidden="true"
          className="size-4 transition-transform duration-200 group-data-[open=true]:rotate-180 motion-reduce:transition-none"
        />
      </button>
      <ol
        className="mt-2 hidden list-none border-l border-rule group-data-[open=true]:block xl:block"
        id="doc-contents"
      >
        {entries.map((entry) => (
          <li key={entry.anchor}>
            <a
              aria-current={entry.anchor === here ? "location" : undefined}
              className={cn(
                "-ml-px flex min-h-9 items-center border-l border-transparent py-1 pr-2 font-ui text-meta text-mute transition-colors hover:text-ink motion-reduce:transition-none",
                "aria-[current=location]:border-accent aria-[current=location]:font-medium aria-[current=location]:text-ink",
                entry.level === 3 ? "pl-7" : "pl-4",
              )}
              href={`#${entry.anchor}`}
              onClick={() => setOpen(false)}
            >
              {entry.label}
            </a>
          </li>
        ))}
      </ol>
    </nav>
  );
}
