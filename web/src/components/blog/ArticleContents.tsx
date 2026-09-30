"use client";

import { useState } from "react";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { cn } from "@/lib/cn";
import type { PostContentsEntry } from "@/lib/post-contents";
import { useReadingMark } from "@/lib/use-reading-mark";

/** ArticleContents lists a long page's headings, folded away on a phone and beside the text on a wide screen. */
export function ArticleContents({
  entries,
  label = "In this article",
}: {
  entries: PostContentsEntry[];
  label?: string;
}) {
  const [open, setOpen] = useState("");
  const here = useReadingMark(entries.map((entry) => entry.anchor));

  const list = (
    <ol className="list-none">
      {entries.map((entry) => (
        <li data-level={entry.level} key={entry.anchor}>
          <a
            aria-current={entry.anchor === here ? "location" : undefined}
            className={cn(
              "flex min-h-control items-center py-1 text-ui text-mute transition-colors duration-80 hover:text-ink motion-reduce:transition-none",
              "aria-[current=location]:font-medium aria-[current=location]:text-accent",
              entry.level === 3 ? "pl-4" : null,
            )}
            href={`#${entry.anchor}`}
            onClick={() => setOpen("")}
          >
            {entry.label}
          </a>
        </li>
      ))}
    </ol>
  );

  return (
    <nav aria-label="Contents" className="font-ui">
      <Accordion
        className="lg:hidden"
        collapsible
        onValueChange={setOpen}
        type="single"
        value={open}
      >
        <AccordionItem value="contents">
          <AccordionTrigger>{label}</AccordionTrigger>
          <AccordionContent className="pb-2 pl-6">{list}</AccordionContent>
        </AccordionItem>
      </Accordion>
      <div className="hidden lg:block">
        <p className="text-meta font-medium text-mute">{label}</p>
        <div className="mt-3">{list}</div>
      </div>
    </nav>
  );
}
