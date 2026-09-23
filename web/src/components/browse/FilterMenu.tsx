"use client";

import { ChevronDown, SlidersHorizontal, X } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { BrowseFilters, BrowsePage } from "@/lib/api/query";
import { chooseFilter } from "@/lib/browse-url";
import { cn } from "@/lib/cn";
import { TYPE_PLURALS } from "@/lib/work-types";
import { CONTROL } from "./BrowseStates";

const ANY = "any";

const ROW = "pointer-fine:min-h-9";

type Facets = BrowsePage["facets"];

function keepOpen(event: Event) {
  event.preventDefault();
}

function chosen(facets: Facets) {
  return facets.flatMap((facet) =>
    facet.options
      .filter((option) => option.selected)
      .map((option) => ({ facet, option })),
  );
}

/** FilterMenu holds every filter the chosen type offers behind one control. */
export function FilterMenu({
  className,
  facets,
  filters,
  navigate,
}: {
  className?: string;
  facets: Facets;
  filters: BrowseFilters;
  navigate: (next: BrowseFilters) => void;
}) {
  const inUse = chosen(facets).length;

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger className={cn(CONTROL, className)}>
        <SlidersHorizontal aria-hidden="true" className="size-4 text-mute" />
        Filters
        {inUse ? (
          <span className="grid h-5 min-w-5 place-items-center rounded-full bg-action px-1.5 text-label text-on-accent tabular-nums">
            {inUse}
          </span>
        ) : null}
        <ChevronDown aria-hidden="true" className="size-4 text-mute" />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-[min(20rem,calc(100vw-2rem))]"
      >
        {facets.length ? null : (
          <p className="px-3 py-2 text-ui text-mute">
            {filters.type
              ? `No filters for ${TYPE_PLURALS[filters.type].toLowerCase()}.`
              : "Pick a type to see its filters."}
          </p>
        )}
        {facets.map((facet, index) => {
          const picked = facet.options.find((option) => option.selected);
          return (
            <DropdownMenuGroup key={facet.key}>
              {index ? <DropdownMenuSeparator /> : null}
              <DropdownMenuLabel className="pb-1 text-meta font-medium text-mute">
                {facet.label}
              </DropdownMenuLabel>
              <DropdownMenuRadioGroup
                onValueChange={(value) =>
                  navigate(
                    chooseFilter(
                      filters,
                      facet.key,
                      value === ANY ? null : value,
                    ),
                  )
                }
                value={picked?.value ?? ANY}
              >
                <DropdownMenuRadioItem
                  className={ROW}
                  onSelect={keepOpen}
                  value={ANY}
                >
                  Any
                </DropdownMenuRadioItem>
                {facet.options.map((option) => (
                  <DropdownMenuRadioItem
                    className={ROW}
                    disabled={option.count === 0 && !option.selected}
                    key={option.value}
                    onSelect={keepOpen}
                    value={option.value}
                  >
                    <span className="flex min-w-0 flex-1 items-baseline justify-between gap-3">
                      <span className="min-w-0 [overflow-wrap:anywhere]">
                        {option.label}
                      </span>
                      <span className="text-meta text-mute tabular-nums">
                        {option.count}
                      </span>
                    </span>
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuGroup>
          );
        })}
        {inUse ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onSelect={() => navigate({ ...filters, facet: undefined })}
            >
              Clear filters
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** ActiveFilters lists the filters in use, each removable on its own. */
export function ActiveFilters({
  facets,
  filters,
  navigate,
}: {
  facets: Facets;
  filters: BrowseFilters;
  navigate: (next: BrowseFilters) => void;
}) {
  const inUse = chosen(facets);
  if (!inUse.length) return null;

  return (
    <ul className="m-0 flex list-none flex-wrap items-center gap-2 p-0 font-ui text-meta">
      {inUse.map(({ facet, option }) => (
        <li key={facet.key}>
          <button
            aria-label={`Remove ${facet.label}: ${option.label}`}
            className="inline-flex min-h-9 max-w-full items-center gap-1.5 rounded-control bg-accent-wash py-1 pr-2 pl-3 text-left font-medium text-ink outline-offset-2 transition-colors duration-200 hover:bg-action hover:text-on-accent motion-reduce:transition-none"
            onClick={() => navigate(chooseFilter(filters, facet.key, null))}
            type="button"
          >
            <span className="min-w-0 [overflow-wrap:anywhere]">
              {facet.label}: {option.label}
            </span>
            <X aria-hidden="true" className="size-3.5 shrink-0" />
          </button>
        </li>
      ))}
      {inUse.length > 1 ? (
        <li>
          <button
            className="inline-flex min-h-9 items-center rounded-control px-2 font-medium text-mute outline-offset-2 hover:text-ink"
            onClick={() => navigate({ ...filters, facet: undefined })}
            type="button"
          >
            Clear filters
          </button>
        </li>
      ) : null}
    </ul>
  );
}
