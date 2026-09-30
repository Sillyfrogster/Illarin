"use client";

import { ChevronDown, SlidersHorizontal, X } from "lucide-react";
import { Button } from "@/components/ui/button";
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
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button className={cn("gap-1.5", className)}>
          <SlidersHorizontal aria-hidden="true" className="text-mute" />
          Filters
          {inUse ? (
            <span className="grid h-5 min-w-5 place-items-center rounded-full bg-action px-1.5 text-label text-on-accent tabular-nums">
              {inUse}
            </span>
          ) : null}
          <ChevronDown aria-hidden="true" className="text-mute" />
        </Button>
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
          <Button
            aria-label={`Remove ${facet.label}: ${option.label}`}
            className="h-auto min-h-control-compact max-w-full py-1 whitespace-normal pr-2 text-left"
            onClick={() => navigate(chooseFilter(filters, facet.key, null))}
            size="compact"
          >
            <span className="min-w-0 [overflow-wrap:anywhere]">
              {facet.label}: {option.label}
            </span>
            <X aria-hidden="true" />
          </Button>
        </li>
      ))}
      {inUse.length > 1 ? (
        <li>
          <Button
            onClick={() => navigate({ ...filters, facet: undefined })}
            size="compact"
            variant="ghost"
          >
            Clear filters
          </Button>
        </li>
      ) : null}
    </ul>
  );
}
