"use client";

import { Check, ChevronDown } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { BrowseFilters, BrowsePage } from "@/lib/api/query";
import { cn } from "@/lib/cn";

const PRESENT = "true";

const ANY = "any";

const chip =
  "inline-flex min-h-9 items-center gap-1.5 rounded-control px-3 font-medium outline-offset-2 transition-colors duration-200 motion-reduce:transition-none";

const on = "bg-action text-on-accent";

const off =
  "bg-deep text-ink hover:bg-accent-wash data-[state=open]:bg-accent-wash disabled:bg-transparent disabled:text-mute disabled:opacity-60";

/** FeatureRow offers what a type's works can hold, such as expressions or a lorebook, as toggles with their counts. */
export function FeatureRow({
  facets,
  filters,
  navigate,
}: {
  facets: BrowsePage["facets"];
  filters: BrowseFilters;
  navigate: (next: BrowseFilters) => void;
}) {
  if (!facets.length) return null;
  const inUse = filters.facet?.length ?? 0;

  function choose(key: string, value: string | null) {
    const kept = (filters.facet ?? []).filter(
      (one) => !one.startsWith(`${key}=`),
    );
    const next = value === null ? kept : [...kept, `${key}=${value}`];
    navigate({ ...filters, facet: next.length ? next : undefined });
  }

  return (
    <div className="-mx-(--gutter) flex items-center gap-x-2 overflow-x-auto px-(--gutter) py-1 font-ui text-ui whitespace-nowrap [scrollbar-width:none] sm:mx-0 sm:flex-wrap sm:gap-x-3 sm:gap-y-2 sm:overflow-visible sm:px-0 [&::-webkit-scrollbar]:hidden [&>*]:shrink-0">
      <span className="mr-1 text-mute">With</span>
      {facets.map((facet) => {
        const present = facet.options.find((one) => one.value === PRESENT);
        if (present && facet.options.length === 2) {
          return (
            <Toggle
              count={present.count}
              key={facet.key}
              label={facet.label}
              onClick={() =>
                choose(facet.key, present.selected ? null : PRESENT)
              }
              selected={present.selected}
            />
          );
        }
        const picked = facet.options.find((one) => one.selected);
        return (
          <DropdownMenu key={facet.key} modal={false}>
            <DropdownMenuTrigger className={cn(chip, picked ? on : off)}>
              {picked ? (
                <Check aria-hidden="true" className="-ml-0.5 size-3.5" />
              ) : null}
              {picked ? `${facet.label}: ${picked.label}` : facet.label}
              <ChevronDown aria-hidden="true" className="size-3.5 opacity-70" />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="start"
              className="w-[min(16rem,calc(100vw-2rem))]"
            >
              <DropdownMenuRadioGroup
                onValueChange={(value) =>
                  choose(facet.key, value === ANY ? null : value)
                }
                value={picked?.value ?? ANY}
              >
                <DropdownMenuRadioItem value={ANY}>
                  Any number
                </DropdownMenuRadioItem>
                {facet.options.map((option) => (
                  <DropdownMenuRadioItem
                    disabled={option.count === 0 && !option.selected}
                    key={option.value}
                    value={option.value}
                  >
                    <span className="flex flex-1 justify-between gap-3">
                      {option.label}
                      <span className="text-meta text-mute tabular-nums">
                        {option.count}
                      </span>
                    </span>
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      })}
      {inUse ? (
        <button
          className="inline-flex min-h-9 items-center rounded-control px-2 font-medium text-mute outline-offset-2 hover:text-ink"
          onClick={() => navigate({ ...filters, facet: undefined })}
          type="button"
        >
          Clear
        </button>
      ) : null}
    </div>
  );
}

function Toggle({
  count,
  label,
  onClick,
  selected,
}: {
  count: number;
  label: string;
  onClick: () => void;
  selected: boolean;
}) {
  return (
    <button
      aria-pressed={selected}
      className={cn(chip, selected ? on : off)}
      disabled={count === 0 && !selected}
      onClick={onClick}
      type="button"
    >
      {selected ? (
        <Check aria-hidden="true" className="-ml-0.5 size-3.5" />
      ) : null}
      {label}
      <span
        className={cn(
          "text-meta tabular-nums",
          selected ? "text-on-accent/75" : "text-mute",
        )}
      >
        {count}
      </span>
    </button>
  );
}
