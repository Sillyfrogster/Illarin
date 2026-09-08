"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { CornerDownLeft, Search } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import type { Asset } from "./data";
import { cn, usePortalTarget } from "./ui";

export type Destination = {
  id: string;
  label: string;
  context: string;
  detail?: string;
  go: () => void;
};

export function destinationsFor(
  asset: Asset,
  go: (blockId: string, elementId: string, itemId?: string) => void,
  openDetails: () => void,
): Destination[] {
  const list: Destination[] = [
    {
      id: "details",
      label: "Asset details",
      context: "Top of the page",
      detail: asset.name,
      go: openDetails,
    },
  ];
  for (const block of asset.blocks)
    for (const element of block.elements) {
      list.push({
        id: `${block.id}:${element.id}`,
        label: element.label,
        context: block.title,
        detail:
          element.type === "prose"
            ? element.text.slice(0, 90)
            : `${element.items.length} ${element.items.length === 1 ? "entry" : "entries"}`,
        go: () => go(block.id, element.id),
      });
      for (const item of element.items)
        list.push({
          id: `${block.id}:${element.id}:${item.id}`,
          label: item.name || "Untitled",
          context: `${block.title} · ${element.label}`,
          detail: item.keys || item.text.slice(0, 90),
          go: () => go(block.id, element.id, item.id),
        });
    }
  return list;
}

/** Adapted from Vengeance UI's Search Modal. See THIRD_PARTY.md. */
export function JumpPalette({
  open,
  onOpenChange,
  destinations,
}: {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  destinations: Destination[];
}) {
  const reduced = useReducedMotion();
  const container = usePortalTarget();
  const [query, setQuery] = useState("");
  const [index, setIndex] = useState(0);
  const list = useRef<HTMLUListElement>(null);

  const matches = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const found = needle
      ? destinations.filter((entry) =>
          `${entry.label} ${entry.context} ${entry.detail ?? ""}`
            .toLowerCase()
            .includes(needle),
        )
      : destinations;
    return found.slice(0, 60);
  }, [destinations, query]);

  useEffect(() => {
    if (open) {
      setQuery("");
      setIndex(0);
    }
  }, [open]);

  useEffect(() => {
    list.current?.children[index]?.scrollIntoView({ block: "nearest" });
  }, [index]);

  function choose(entry?: Destination) {
    if (!entry) return;
    onOpenChange(false);
    entry.go();
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <AnimatePresence>
        {open && (
          <Dialog.Portal forceMount container={container}>
            <Dialog.Overlay asChild forceMount>
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: reduced ? 0 : 0.2 }}
                className="ws:fixed ws:inset-0 ws:z-80 ws:bg-black/40 ws:backdrop-blur-[4px]"
              />
            </Dialog.Overlay>
            <Dialog.Content asChild forceMount>
              <motion.div
                initial={{
                  opacity: 0,
                  y: reduced ? 0 : -14,
                  scale: reduced ? 1 : 0.98,
                }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{
                  opacity: 0,
                  y: reduced ? 0 : -10,
                  scale: reduced ? 1 : 0.98,
                }}
                transition={{
                  duration: reduced ? 0 : 0.24,
                  ease: [0.22, 1, 0.36, 1],
                }}
                onKeyDown={(event) => {
                  if (event.key === "ArrowDown") {
                    event.preventDefault();
                    setIndex((i) => Math.min(i + 1, matches.length - 1));
                  } else if (event.key === "ArrowUp") {
                    event.preventDefault();
                    setIndex((i) => Math.max(i - 1, 0));
                  } else if (event.key === "Enter") {
                    event.preventDefault();
                    choose(matches[index]);
                  }
                }}
                className="ws:fixed ws:inset-x-0 ws:top-[12dvh] ws:z-90 ws:mx-auto ws:flex ws:max-h-[70dvh] ws:w-[min(40rem,92vw)] ws:flex-col ws:overflow-hidden ws:rounded-[22px] ws:bg-card ws:text-ink ws:shadow-[var(--w-lift)]"
              >
                <div
                  aria-hidden="true"
                  className="w-spectral-rule ws:shrink-0"
                />
                <Dialog.Title className="ws:sr-only">
                  Go to page content
                </Dialog.Title>
                <Dialog.Description className="ws:sr-only">
                  Filter every block, element and entry on this page, then press
                  Enter to open it for writing.
                </Dialog.Description>
                <div className="ws:relative ws:shrink-0">
                  <Search className="ws:pointer-events-none ws:absolute ws:top-1/2 ws:left-5 ws:size-4 ws:-translate-y-1/2 ws:text-mute" />
                  <input
                    // biome-ignore lint/a11y/noAutofocus: a command palette opens on its input
                    autoFocus
                    value={query}
                    onChange={(e) => {
                      setQuery(e.target.value);
                      setIndex(0);
                    }}
                    placeholder="Go to anything on this page"
                    aria-label="Filter page content"
                    className="ws:h-16 ws:w-full ws:bg-transparent ws:pl-12 ws:pr-5 ws:text-base ws:text-ink ws:outline-none ws:placeholder:text-mute"
                  />
                </div>
                <div
                  aria-hidden="true"
                  className="ws:h-px ws:shrink-0 ws:bg-hairline"
                />
                <ul
                  ref={list}
                  aria-label="Page content"
                  className="ws:min-h-0 ws:flex-1 ws:overflow-y-auto ws:overscroll-contain ws:p-2"
                >
                  {matches.map((entry, position) => (
                    <li key={entry.id}>
                      <button
                        type="button"
                        aria-current={position === index ? "true" : undefined}
                        onMouseEnter={() => setIndex(position)}
                        onClick={() => choose(entry)}
                        className={cn(
                          "ws:flex ws:min-h-12 ws:w-full ws:items-center ws:gap-4 ws:rounded-xl ws:px-3.5 ws:text-left",
                          position === index && "ws:bg-ink/8",
                        )}
                      >
                        <span className="ws:min-w-0 ws:flex-1">
                          <span className="ws:block ws:truncate ws:text-sm ws:font-semibold">
                            {entry.label}
                          </span>
                          <span className="ws:block ws:truncate ws:text-xs ws:text-mute">
                            {entry.context}
                            {entry.detail ? ` · ${entry.detail}` : ""}
                          </span>
                        </span>
                        {position === index && (
                          <CornerDownLeft className="ws:size-3.5 ws:shrink-0 ws:text-mute" />
                        )}
                      </button>
                    </li>
                  ))}
                  {!matches.length && (
                    <li className="ws:px-4 ws:py-6 ws:text-sm ws:text-mute">
                      Nothing on this page matches “{query}”.
                    </li>
                  )}
                </ul>
              </motion.div>
            </Dialog.Content>
          </Dialog.Portal>
        )}
      </AnimatePresence>
    </Dialog.Root>
  );
}
