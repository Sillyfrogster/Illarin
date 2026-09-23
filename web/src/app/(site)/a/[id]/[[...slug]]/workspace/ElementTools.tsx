"use client";

import { ArrowDown, ArrowUp, SquarePen, Trash2 } from "lucide-react";
import type { WorkBlock, WorkElement } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { editsInTheRail } from "@/lib/page-arrangement";
import { moveElement, removeElement } from "./composition";
import { useWorkspace } from "./state";

const TOOL =
  "inline-flex min-h-9 shrink-0 items-center gap-1.5 rounded-control px-2 text-meta font-medium text-mute outline-offset-3 hover:bg-plane hover:text-ink disabled:opacity-35 disabled:hover:bg-transparent";

export function ElementTools({
  block,
  element,
}: {
  block: WorkBlock;
  element: WorkElement;
}) {
  const workspace = useWorkspace();
  const position = block.elements.findIndex((item) => item.id === element.id);
  const total = block.elements.length;
  const name = element.label || "this content";

  function move(to: number) {
    workspace.writeBlock(moveElement(block, element.id, to));
    workspace.say(`${name} moved to ${to + 1} of ${total}.`);
  }

  return (
    <div
      aria-label={`${name} controls`}
      className="flex shrink-0 items-center gap-0.5 rounded-control bg-deep p-0.5 opacity-0 transition-opacity duration-200 group-focus-within/element:opacity-100 group-hover/element:opacity-100 motion-reduce:transition-none max-md:opacity-100"
      data-measurement-ignore
      role="toolbar"
    >
      {editsInTheRail(element.type) ? (
        <button
          className={TOOL}
          onClick={() =>
            workspace.openPane({
              blockId: block.id,
              elementId: element.id,
              kind: "element",
            })
          }
          type="button"
        >
          <SquarePen aria-hidden="true" size={14} />
          Edit
        </button>
      ) : null}
      {total > 1 ? (
        <>
          <button
            aria-label={`Move ${name} earlier, now ${position + 1} of ${total}`}
            className={cn(TOOL, "px-1.5")}
            disabled={position === 0}
            onClick={() => move(position - 1)}
            type="button"
          >
            <ArrowUp aria-hidden="true" size={14} />
          </button>
          <button
            aria-label={`Move ${name} later, now ${position + 1} of ${total}`}
            className={cn(TOOL, "px-1.5")}
            disabled={position === total - 1}
            onClick={() => move(position + 1)}
            type="button"
          >
            <ArrowDown aria-hidden="true" size={14} />
          </button>
        </>
      ) : null}
      {element.pinned ? null : (
        <button
          aria-label={`Remove ${name} from ${block.title}`}
          className={cn(TOOL, "px-1.5 hover:text-stop")}
          onClick={() => {
            workspace.writeBlock(removeElement(block, element.id));
            workspace.say(`${name} removed from “${block.title}”.`);
          }}
          type="button"
        >
          <Trash2 aria-hidden="true" size={14} />
        </button>
      )}
    </div>
  );
}
