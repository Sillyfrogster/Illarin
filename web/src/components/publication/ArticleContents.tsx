"use client";

import { ChevronDown } from "lucide-react";
import { useEffect, useState } from "react";
import { cn } from "@/lib/cn";
import type { PostContentsEntry } from "@/lib/post-contents";

/** How far down the window a heading has to pass before the contents call its section the one being read. */
const READING_LINE = 0.18;

/** The outline a post's own headings make, in the margin on a wide screen and a disclosure on a narrow one. */
export function ArticleContents({ entries }: { entries: PostContentsEntry[] }) {
  const [open, setOpen] = useState(false);
  const [here, setHere] = useState("");

  useEffect(() => {
    const marks = entries
      .map((entry) => document.getElementById(entry.anchor))
      .filter((mark): mark is HTMLElement => mark !== null);
    if (marks.length === 0) return;
    let asked = 0;
    const settle = () => {
      asked = 0;
      const line = window.innerHeight * READING_LINE;
      let reached = "";
      for (const mark of marks) {
        if (mark.getBoundingClientRect().top <= line) reached = mark.id;
      }
      setHere(reached);
    };
    const ask = () => {
      if (asked === 0) asked = requestAnimationFrame(settle);
    };
    settle();
    window.addEventListener("scroll", ask, { passive: true });
    window.addEventListener("resize", ask);
    return () => {
      if (asked !== 0) cancelAnimationFrame(asked);
      window.removeEventListener("scroll", ask);
      window.removeEventListener("resize", ask);
    };
  }, [entries]);

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
