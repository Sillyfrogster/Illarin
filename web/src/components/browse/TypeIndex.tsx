"use client";

import {
  LayoutGroup,
  motion,
  type TargetAndTransition,
  useReducedMotion,
} from "framer-motion";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { Scroller } from "@/components/ui/scroller";
import type { BrowseFilters, BrowsePage, BrowseType } from "@/lib/api/query";
import { buildBrowseHref } from "@/lib/browse-url";
import { cn } from "@/lib/cn";
import { TYPE_PLURALS, WORK_TYPES } from "@/lib/work-types";

const FLIGHT = {
  type: "spring",
  stiffness: 140,
  damping: 18,
  mass: 0.9,
} as const;

const WING =
  "h-full w-1/2 bg-[url(/landing/flight/butterfly.webp)] bg-size-[200%_100%] bg-no-repeat";

/** TypeIndex names every type browse holds as the page's heading, with the butterfly resting on the chosen one. */
export function TypeIndex({
  basePath,
  compact = false,
  filters,
  navigate,
  overview,
}: {
  basePath: string;
  compact?: boolean;
  filters: BrowseFilters;
  navigate: (next: BrowseFilters) => void;
  overview: BrowsePage | undefined;
}) {
  const [chosen, setChosen] = useState(filters.type ?? "all");
  const index = useRef<HTMLElement>(null);
  useEffect(() => setChosen(filters.type ?? "all"), [filters.type]);
  useEffect(() => {
    index.current
      ?.querySelector(`[data-type="${chosen}"]`)
      ?.scrollIntoView({ block: "nearest", inline: "nearest" });
  }, [chosen]);

  const entries: {
    key: string;
    type?: BrowseType;
    label: string;
    count?: number;
  }[] = [
    { key: "all", label: "Everything", count: overview?.allTypes },
    ...WORK_TYPES.flatMap((type) => {
      const held = overview?.types.find((one) => one.value === type);
      return held
        ? [{ key: type, type, label: TYPE_PLURALS[type], count: held.count }]
        : [];
    }),
  ];

  return (
    <nav aria-label="Type" className="-ml-4 min-w-0" ref={index}>
      <LayoutGroup id="browse-type">
        <Scroller buttonClassName="bottom-0.5">
          <ul
            className={cn(
              "m-0 flex w-max list-none gap-x-7 p-0 pt-6 pr-6 pl-4 sm:gap-x-8",
              compact ? "text-section" : "text-[clamp(1.625rem,2.3vw,2.25rem)]",
            )}
          >
            {entries.map((entry) => {
              const here = chosen === entry.key;
              const empty = entry.count === 0 && !here;
              return (
                <li className="shrink-0" key={entry.key}>
                  <Link
                    aria-current={here ? "page" : undefined}
                    data-type={entry.key}
                    className={cn(
                      "relative flex items-start rounded-control font-display leading-[1.15] font-medium tracking-[-0.03em] whitespace-nowrap outline-offset-4 transition-colors duration-300 motion-reduce:transition-none",
                      here ? "text-ink" : "text-mute hover:text-ink",
                      empty && "opacity-45",
                    )}
                    href={buildBrowseHref(
                      { ...filters, type: entry.type, facet: undefined },
                      basePath,
                    )}
                    onClick={(event) => {
                      event.preventDefault();
                      setChosen(entry.key);
                      navigate({
                        ...filters,
                        type: entry.type,
                        facet: undefined,
                      });
                    }}
                  >
                    {entry.label}
                    {entry.count === undefined ? null : (
                      <span
                        className={cn(
                          "ml-1 font-ui text-meta font-medium tracking-normal tabular-nums",
                          here ? "text-accent" : "text-mute",
                        )}
                      >
                        <span className="sr-only">, </span>
                        {entry.count}
                        <span className="sr-only">
                          {entry.count === 1 ? " work" : " works"}
                        </span>
                      </span>
                    )}
                    {here ? <Butterfly landing={entry.key} /> : null}
                  </Link>
                </li>
              );
            })}
          </ul>
        </Scroller>
      </LayoutGroup>
    </nav>
  );
}

function Butterfly({ landing }: { landing: string }) {
  const still = useReducedMotion();
  const beat: TargetAndTransition | undefined = still
    ? undefined
    : {
        scaleX: [1, 0.2, 1, 0.3, 1],
        transition: {
          duration: 0.7,
          ease: "easeInOut",
          repeat: Number.POSITIVE_INFINITY,
          repeatDelay: 6,
        },
      };

  return (
    <motion.span
      aria-hidden="true"
      className="pointer-events-none absolute -top-5.5 -left-4 flex h-6 w-9 -rotate-[18deg] drop-shadow-[0_3px_10px_rgb(167_120_255/0.5)]"
      layoutId="butterfly"
      transition={still ? { duration: 0 } : FLIGHT}
    >
      <motion.span
        animate={beat}
        className={cn(WING, "origin-right bg-left")}
        key={`left-${landing}`}
      />
      <motion.span
        animate={beat}
        className={cn(WING, "origin-left bg-right")}
        key={`right-${landing}`}
      />
    </motion.span>
  );
}
