"use client";

import {
  Bold,
  ChevronDown,
  Code,
  Italic,
  Link,
  List,
  ListOrdered,
  TextQuote,
} from "lucide-react";
import { type ComponentType, useId, useState } from "react";
import { cn } from "@/lib/cn";
import type { MarkdownAction } from "@/lib/markdown-edit";

const TOOL =
  "inline-flex size-11 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-plane hover:text-ink md:size-9";

const SUMMARY =
  "inline-flex min-h-9 items-center gap-1 rounded-control font-ui text-label text-mute outline-offset-3 hover:text-ink";

type Tool = {
  action: MarkdownAction;
  icon: ComponentType<{ "aria-hidden": "true"; size: number }>;
  label: string;
};

const TOOLS: Tool[] = [
  { action: "bold", icon: Bold, label: "Bold" },
  { action: "italic", icon: Italic, label: "Italic" },
  { action: "code", icon: Code, label: "Inline code" },
  { action: "link", icon: Link, label: "Link" },
  { action: "bullet", icon: List, label: "Bulleted list" },
  { action: "numbered", icon: ListOrdered, label: "Numbered list" },
  { action: "quote", icon: TextQuote, label: "Block quote" },
];

/** Offers the page markdown a rich field accepts, so a creator does not have to know the markers */
export function FormattingBar({
  apply,
}: {
  apply: (action: MarkdownAction) => void;
}) {
  const [open, setOpen] = useState(false);
  const detail = useId();

  return (
    <div className="mb-3">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <div
          aria-label="Formatting"
          className="flex shrink-0 items-center gap-0.5 rounded-plate bg-deep p-0.5"
          data-measurement-ignore
          role="toolbar"
        >
          {TOOLS.map(({ action, icon: Icon, label }) => (
            <button
              aria-label={label}
              className={TOOL}
              key={action}
              onClick={() => apply(action)}
              onMouseDown={(event) => event.preventDefault()}
              type="button"
            >
              <Icon aria-hidden="true" size={15} />
            </button>
          ))}
        </div>
        <button
          aria-controls={detail}
          aria-expanded={open}
          className={SUMMARY}
          data-measurement-ignore
          onClick={() => setOpen((shown) => !shown)}
          onMouseDown={(event) => event.preventDefault()}
          type="button"
        >
          Page markdown works here
          <ChevronDown
            aria-hidden="true"
            className={cn(
              "transition-transform duration-200 motion-reduce:transition-none",
              open && "rotate-180",
            )}
            size={13}
          />
        </button>
      </div>
      {open ? (
        <dl
          className="mt-2 grid gap-x-3 gap-y-1 rounded-control bg-deep p-3 font-ui text-label text-mute [grid-template-columns:fit-content(6rem)_minmax(0,1fr)]"
          data-measurement-ignore
          id={detail}
        >
          <dt className="font-medium">Takes</dt>
          <dd>
            Emphasis, links, inline code, headings, lists, block quotes, fenced
            code and tables.
          </dd>
          <dt className="font-medium">Drops</dt>
          <dd>
            Callouts, dividers and images. HTML is reduced to the words inside
            it, and a note says once where anything was dropped.
          </dd>
          <dt className="font-medium">Keeps</dt>
          <dd>A download carries your text exactly as you typed it.</dd>
        </dl>
      ) : null}
    </div>
  );
}
