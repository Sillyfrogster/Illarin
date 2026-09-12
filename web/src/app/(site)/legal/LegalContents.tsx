"use client";

import { ChevronDown } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/cn";
import { clauseAnchor } from "@/lib/legal-documents";
import { useReadingMark } from "@/lib/use-reading-mark";
import type { LegalClause } from "./LegalPage";

export function LegalContents({ clauses }: { clauses: LegalClause[] }) {
  const [open, setOpen] = useState(false);
  const anchors = clauses.map((clause) => clauseAnchor(clause.heading));
  const here = useReadingMark(anchors);

  return (
    <nav aria-label="Contents" className="group" data-open={open}>
      <p
        aria-hidden="true"
        className="hidden font-ui text-meta font-medium text-mute lg:block"
      >
        In this document
      </p>
      <button
        aria-controls="legal-contents"
        aria-expanded={open}
        className="flex min-h-11 w-full items-center justify-between gap-3 rounded-control bg-deep px-4 font-ui text-ui font-medium text-ink outline-offset-3 lg:hidden"
        onClick={() => setOpen(!open)}
        type="button"
      >
        In this document
        <ChevronDown
          aria-hidden="true"
          className="size-4 transition-transform duration-200 group-data-[open=true]:rotate-180 motion-reduce:transition-none"
        />
      </button>
      <ol
        className="mt-2 hidden list-none group-data-[open=true]:block lg:mt-2 lg:block"
        id="legal-contents"
      >
        {clauses.map((clause, index) => (
          <li key={clause.heading}>
            <a
              aria-current={anchors[index] === here ? "location" : undefined}
              className={cn(
                "flex min-h-11 items-center py-1 font-ui text-meta text-mute transition-colors hover:text-ink motion-reduce:transition-none",
                "aria-[current=location]:font-medium aria-[current=location]:text-accent",
              )}
              href={`#${anchors[index]}`}
              onClick={() => setOpen(false)}
            >
              {clause.heading}
            </a>
          </li>
        ))}
      </ol>
    </nav>
  );
}
