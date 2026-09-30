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
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/cn";
import type { MarkdownAction } from "@/lib/markdown-edit";

type Tool = {
  action: MarkdownAction;
  icon: ComponentType<{ "aria-hidden": "true" }>;
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
            <Tooltip content={label} key={action}>
              <Button
                aria-label={label}
                onClick={() => apply(action)}
                onMouseDown={(event) => event.preventDefault()}
                size="icon-compact"
                variant="ghost"
              >
                <Icon aria-hidden="true" />
              </Button>
            </Tooltip>
          ))}
        </div>
        <Button
          aria-controls={detail}
          aria-expanded={open}
          data-measurement-ignore
          onClick={() => setOpen((shown) => !shown)}
          onMouseDown={(event) => event.preventDefault()}
          size="compact"
          variant="ghost"
        >
          Markdown works here
          <ChevronDown
            aria-hidden="true"
            className={cn(
              "transition-transform duration-160 motion-reduce:transition-none",
              open && "rotate-180",
            )}
          />
        </Button>
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
