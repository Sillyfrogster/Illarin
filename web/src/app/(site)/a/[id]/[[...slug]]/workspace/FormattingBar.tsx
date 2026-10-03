"use client";

import {
  Bold,
  CircleHelp,
  Code,
  Italic,
  Link,
  List,
  ListOrdered,
  TextQuote,
} from "lucide-react";
import type { ComponentType } from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Tooltip } from "@/components/ui/tooltip";
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

/** Floats the markdown a rich field accepts above the text being written, so a creator does not have to know the markers and nothing on the page moves */
export function FormattingBar({
  apply,
}: {
  apply: (action: MarkdownAction) => void;
}) {
  return (
    <div
      aria-label="Formatting"
      className="absolute bottom-2 left-0 flex animate-pop items-center gap-0.5 rounded-control bg-plane p-1 shadow-popover ring-1 ring-ink/8"
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
      <span aria-hidden="true" className="mx-1 h-5 w-px bg-rule" />
      <Popover>
        <Tooltip content="What markdown does here">
          <PopoverTrigger asChild>
            <Button
              aria-label="What markdown does here"
              onMouseDown={(event) => event.preventDefault()}
              size="icon-compact"
              variant="ghost"
            >
              <CircleHelp aria-hidden="true" />
            </Button>
          </PopoverTrigger>
        </Tooltip>
        <PopoverContent
          align="start"
          className="w-[min(24rem,calc(100vw-2rem))]"
          onCloseAutoFocus={(event) => event.preventDefault()}
          onOpenAutoFocus={(event) => event.preventDefault()}
        >
          <dl className="grid gap-x-3 gap-y-2 text-meta text-mute [grid-template-columns:fit-content(5rem)_minmax(0,1fr)]">
            <dt className="font-medium text-ink">Takes</dt>
            <dd>
              Emphasis, links, inline code, headings, lists, block quotes,
              fenced code and tables.
            </dd>
            <dt className="font-medium text-ink">Drops</dt>
            <dd>
              Callouts, dividers and images. HTML is reduced to the words inside
              it, and a note says once where anything was dropped.
            </dd>
            <dt className="font-medium text-ink">Keeps</dt>
            <dd>A download carries your text exactly as you typed it.</dd>
          </dl>
        </PopoverContent>
      </Popover>
    </div>
  );
}
