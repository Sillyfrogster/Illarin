"use client";

import { Search } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import type { WorkBlock } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { elementLabel } from "@/lib/element-label";

export type Destination = {
  blockId: string;
  elementId?: string;
  id: string;
  label: string;
  where: string;
};

export function destinationsIn(blocks: WorkBlock[]): Destination[] {
  return blocks.flatMap((block) => [
    {
      blockId: block.id,
      id: block.id,
      label: block.title,
      where: "Block",
    },
    ...block.elements.map((element) => ({
      blockId: block.id,
      elementId: element.id,
      id: element.id,
      label:
        elementLabel(element, {
          elements: block.elements.length,
          title: block.title,
        }) ||
        element.label ||
        "Content",
      where: block.title,
    })),
  ]);
}

export function JumpPalette({
  destinations,
  onClose,
  onGo,
}: {
  destinations: Destination[];
  onClose: () => void;
  onGo: (destination: Destination) => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [query, setQuery] = useState("");
  const [at, setAt] = useState(0);

  const matches = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return destinations;
    return destinations.filter((destination) =>
      `${destination.label} ${destination.where}`
        .toLowerCase()
        .includes(needle),
    );
  }, [destinations, query]);

  useEffect(() => {
    dialog.current?.showModal();
  }, []);

  useEffect(() => setAt(0), []);

  function choose(destination: Destination | undefined) {
    if (!destination) return;
    dialog.current?.close();
    onGo(destination);
  }

  return (
    <dialog
      aria-label="Go to content"
      className="m-0 mx-auto mt-[12vh] w-[min(34rem,calc(100vw-2rem))] rounded-plate bg-plane p-0 text-ink shadow-popover backdrop:bg-ink/45"
      onCancel={(event) => {
        event.preventDefault();
        dialog.current?.close();
        onClose();
      }}
      onClose={onClose}
      ref={dialog}
    >
      <div className="flex items-center gap-3 border-b border-rule px-4">
        <Search aria-hidden="true" className="shrink-0 text-mute" size={17} />
        <input
          aria-label="Search this page"
          autoFocus
          className="min-h-14 w-full border-0 bg-transparent text-ui text-ink outline-none placeholder:text-mute"
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "ArrowDown") {
              event.preventDefault();
              setAt((index) => Math.min(index + 1, matches.length - 1));
            }
            if (event.key === "ArrowUp") {
              event.preventDefault();
              setAt((index) => Math.max(index - 1, 0));
            }
            if (event.key === "Enter") {
              event.preventDefault();
              choose(matches[at]);
            }
          }}
          placeholder="Go to a block or a piece of writing"
          value={query}
        />
      </div>
      <ul className="max-h-[52vh] list-none overflow-y-auto p-2">
        {matches.length === 0 ? (
          <li className="px-3 py-6 text-center text-meta text-mute">
            Nothing on this page matches that.
          </li>
        ) : null}
        {matches.map((destination, index) => (
          <li key={`${destination.blockId}:${destination.id}`}>
            <button
              className={cn(
                "flex min-h-11 w-full items-baseline justify-between gap-4 rounded-control px-3 py-2 text-left outline-offset-3 hover:bg-deep",
                index === at && "bg-deep",
              )}
              onClick={() => choose(destination)}
              onMouseEnter={() => setAt(index)}
              type="button"
            >
              <span className="min-w-0 truncate text-ui text-ink">
                {destination.label}
              </span>
              <span className="shrink-0 text-label text-mute">
                {destination.where}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </dialog>
  );
}
