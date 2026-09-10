"use client";

import { ChevronDown } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/cn";
import type { PostContentsEntry } from "@/lib/post-contents";
import { useReadingMark } from "@/lib/use-reading-mark";

export function ArticleContents({ entries }: { entries: PostContentsEntry[] }) {
  const [open, setOpen] = useState(false);
  const here = useReadingMark(entries.map((entry) => entry.anchor));

  return (
    <nav aria-label="Contents" className="group" data-open={open}>
      <p aria-hidden="true" className="hidden text-meta font-medium lg:block">
        In this article
      </p>
      <button
        aria-controls="article-contents"
        aria-expanded={open}
        className="flex min-h-11 w-full items-center justify-between gap-3 rounded-control bg-deep px-4 text-ui font-medium text-ink lg:hidden"
        onClick={() => setOpen(!open)}
        type="button"
      >
        In this article
        <ChevronDown
          aria-hidden="true"
          className="size-4 transition-transform duration-200 group-data-[open=true]:rotate-180 motion-reduce:transition-none"
        />
      </button>
      <ol
        className="mt-2 hidden list-none group-data-[open=true]:block lg:mt-3 lg:block"
        id="article-contents"
      >
        {entries.map((entry) => (
          <li data-level={entry.level} key={entry.anchor}>
            <a
              aria-current={entry.anchor === here ? "location" : undefined}
              className={cn(
                "flex min-h-11 items-center py-1 text-ui text-mute transition-colors hover:text-ink",
                "aria-[current=location]:font-medium aria-[current=location]:text-accent",
                entry.level === 3 ? "pl-4" : null,
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
