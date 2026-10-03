"use client";

import { ArrowDown, ArrowUp, SquarePen, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import type { WorkBlock, WorkElement } from "@/lib/api/query";
import { editsInTheRail } from "@/lib/page-arrangement";
import { moveElement, removeElement } from "./composition";
import { useWorkspaceActions } from "./state";

export function ElementTools({
  block,
  element,
}: {
  block: WorkBlock;
  element: WorkElement;
}) {
  const workspace = useWorkspaceActions();
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
      className="flex shrink-0 items-center gap-0.5 rounded-control bg-deep p-0.5 opacity-0 transition-opacity duration-160 group-focus-within/element:opacity-100 group-hover/element:opacity-100 motion-reduce:transition-none max-md:opacity-100"
      data-measurement-ignore
      role="toolbar"
    >
      {editsInTheRail(element.type) ? (
        <Button
          onClick={() =>
            workspace.openPane({
              blockId: block.id,
              elementId: element.id,
              kind: "element",
            })
          }
          size="compact"
          variant="ghost"
        >
          <SquarePen aria-hidden="true" />
          Edit
        </Button>
      ) : null}
      {total > 1 ? (
        <>
          <Tooltip content="Move earlier">
            <Button
              aria-label={`Move ${name} earlier, now ${position + 1} of ${total}`}
              disabled={position === 0}
              onClick={() => move(position - 1)}
              size="icon-compact"
              variant="ghost"
            >
              <ArrowUp aria-hidden="true" />
            </Button>
          </Tooltip>
          <Tooltip content="Move later">
            <Button
              aria-label={`Move ${name} later, now ${position + 1} of ${total}`}
              disabled={position === total - 1}
              onClick={() => move(position + 1)}
              size="icon-compact"
              variant="ghost"
            >
              <ArrowDown aria-hidden="true" />
            </Button>
          </Tooltip>
        </>
      ) : null}
      {element.pinned ? null : (
        <Tooltip content="Remove">
          <Button
            aria-label={`Remove ${name} from ${block.title}`}
            className="hover:text-stop"
            onClick={() => {
              workspace.writeBlock(removeElement(block, element.id));
              workspace.say(`${name} removed from “${block.title}”.`);
            }}
            size="icon-compact"
            variant="ghost"
          >
            <Trash2 aria-hidden="true" />
          </Button>
        </Tooltip>
      )}
    </div>
  );
}
