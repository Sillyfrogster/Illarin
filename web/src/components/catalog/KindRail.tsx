"use client";

import Link from "next/link";
import { useState } from "react";
import { TravellingHighlight } from "@/components/ui/travelling-highlight";
import type { BrowseFilters, BrowseKind } from "@/lib/api/query";
import { buildBrowseHref } from "@/lib/browse-url";
import { cn } from "@/lib/cn";
import { KindMark } from "./KindMark";

const KINDS: { key: string; kind?: BrowseKind; label: string }[] = [
  { key: "all", label: "Everything" },
  { key: "character", kind: "character", label: "Characters" },
  { key: "lorebook", kind: "lorebook", label: "Lorebooks" },
  { key: "preset", kind: "preset", label: "Presets" },
  { key: "theme", kind: "theme", label: "Themes" },
  { key: "pack", kind: "pack", label: "Packs" },
  { key: "extension", kind: "extension", label: "Extensions" },
];

export function KindRail({
  basePath,
  filters,
  navigate,
}: {
  basePath: string;
  filters: BrowseFilters;
  navigate: (next: BrowseFilters) => void;
}) {
  const chosen = filters.kind ?? "all";
  const [lit, setLit] = useState(chosen);

  return (
    <nav aria-label="Kind" className="min-w-0">
      <div className="overflow-x-auto p-1 [scrollbar-width:none] max-md:[mask-image:linear-gradient(to_right,#000_calc(100%-3rem),transparent)] [&::-webkit-scrollbar]:hidden">
        <TravellingHighlight
          chosen={chosen}
          className="flex w-max gap-1 max-md:pr-10"
          onLit={setLit}
        >
          {KINDS.map((one) => (
            <Link
              aria-current={chosen === one.key ? "page" : undefined}
              className={cn(
                "flex min-h-11 items-center gap-2 rounded-control px-4 font-ui text-ui font-medium whitespace-nowrap outline-offset-2 transition-colors duration-200 motion-reduce:transition-none",
                lit === one.key ? "text-on-accent" : "text-mute",
              )}
              data-cell={one.key}
              href={buildBrowseHref(
                { ...filters, kind: one.kind, facet: undefined },
                basePath,
              )}
              key={one.key}
              onClick={(event) => {
                event.preventDefault();
                navigate({ ...filters, kind: one.kind, facet: undefined });
              }}
            >
              {one.kind ? (
                <KindMark className="size-4" kind={one.kind} />
              ) : null}
              {one.label}
            </Link>
          ))}
        </TravellingHighlight>
      </div>
    </nav>
  );
}
