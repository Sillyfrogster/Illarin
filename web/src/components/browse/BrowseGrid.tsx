"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import {
  memo,
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import type { BrowseWork, NsfwPreference } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { workDisplayName } from "@/lib/work-name";
import { OpenPanel } from "./OpenPanel";
import { usePrefetchPeek } from "./peek";
import { TileArt, TileText } from "./WorkTile";

export const TILES =
  "m-0 grid list-none grid-cols-2 gap-3 p-0 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4 xl:grid-cols-5";

const SETTLE = [0.22, 1, 0.36, 1] as const;

// A small fixed lean per work, so dropped cards fall at different angles without shuffling on every render.
function lean(id: string) {
  let hash = 0;
  for (const character of id)
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) % 997;
  return (hash % 13) - 6;
}

function useColumns() {
  const list = useRef<HTMLUListElement>(null);
  const [columns, setColumns] = useState(1);
  useLayoutEffect(() => {
    const here = list.current;
    if (!here) return;
    const measure = () =>
      setColumns(
        getComputedStyle(here).gridTemplateColumns.split(" ").filter(Boolean)
          .length || 1,
      );
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(here);
    return () => observer.disconnect();
  }, []);
  return { list, columns };
}

/** BrowseGrid shows the works as cards: a narrowed choice lets the works that no longer match fall out, and a pressed card opens under its row. */
export function BrowseGrid({
  preference,
  works,
}: {
  preference: NsfwPreference;
  works: BrowseWork[];
}) {
  const still = useReducedMotion();
  const { list, columns } = useColumns();
  const [openId, setOpenId] = useState<string | null>(null);
  const panel = useId();
  const prefetch = usePrefetchPeek();
  const toggle = useCallback(
    (id: string) => setOpenId((now) => (now === id ? null : id)),
    [],
  );
  const found = openId ? works.findIndex((work) => work.id === openId) : -1;
  const at = found >= 0 ? found : null;
  const rowEnd =
    at === null
      ? -1
      : Math.min(
          works.length - 1,
          Math.floor(at / columns) * columns + columns - 1,
        );

  useEffect(() => {
    if (at === null) return;
    function key(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (target?.closest("input, textarea, [role=menu], [role=listbox]"))
        return;
      if (event.key === "Escape") setOpenId(null);
      if (event.key === "ArrowRight" && at !== null && at < works.length - 1)
        setOpenId(works[at + 1].id);
      if (event.key === "ArrowLeft" && at !== null && at > 0)
        setOpenId(works[at - 1].id);
    }
    window.addEventListener("keydown", key);
    return () => window.removeEventListener("keydown", key);
  }, [at, works]);

  return (
    <ul className={TILES} ref={list}>
      <AnimatePresence initial={false} mode="popLayout">
        {works.flatMap((work, index) => {
          const card = (
            <motion.li
              animate={{ opacity: 1, y: 0, scale: 1, rotate: 0 }}
              className="flex"
              exit={
                still
                  ? { opacity: 0 }
                  : {
                      opacity: 0,
                      y: 90,
                      scale: 0.86,
                      rotate: lean(work.id),
                      transition: {
                        duration: 0.42,
                        ease: [0.55, 0, 0.75, 0.2],
                      },
                    }
              }
              initial={
                still ? { opacity: 0 } : { opacity: 0, y: -28, scale: 0.94 }
              }
              key={work.id}
              layout={!still}
              transition={{
                duration: 0.5,
                ease: SETTLE,
                delay: still ? 0 : Math.min(index, 12) * 0.018,
              }}
            >
              <Tile
                controls={index === at ? panel : undefined}
                eager={index < 5}
                onOpen={toggle}
                onPoint={prefetch}
                open={index === at}
                preference={preference}
                work={work}
              />
            </motion.li>
          );
          if (index !== rowEnd || at === null) return [card];
          return [
            card,
            <OpenPanel
              id={panel}
              key={`panel-${Math.floor(at / columns)}`}
              next={
                at < works.length - 1 ? () => setOpenId(works[at + 1].id) : null
              }
              onClose={() => setOpenId(null)}
              preference={preference}
              previous={at > 0 ? () => setOpenId(works[at - 1].id) : null}
              work={works[at]}
            />,
          ];
        })}
      </AnimatePresence>
    </ul>
  );
}

const Tile = memo(function Tile({
  controls,
  eager,
  onOpen,
  onPoint,
  open,
  preference,
  work,
}: {
  controls: string | undefined;
  eager: boolean;
  onOpen: (id: string) => void;
  onPoint: (id: string) => void;
  open: boolean;
  preference: NsfwPreference;
  work: BrowseWork;
}) {
  return (
    <div
      className={cn(
        "group/tile relative flex w-full min-w-0 flex-col rounded-card bg-plane p-2 shadow-card ring-1 ring-ink/8 focus-within:ring-accent",
        open && "ring-accent",
      )}
    >
      <TileArt eager={eager} preference={preference} work={work} />
      <TileText work={work} />
      <button
        aria-controls={controls}
        aria-expanded={open}
        aria-label={`Open ${workDisplayName(work.name)}`}
        className="absolute inset-0 cursor-pointer rounded-card outline-none"
        onClick={() => onOpen(work.id)}
        onPointerEnter={() => onPoint(work.id)}
        type="button"
      />
    </div>
  );
});
