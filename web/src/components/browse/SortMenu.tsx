"use client";

import { ArrowDownWideNarrow, ChevronDown } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { BrowseFilters, BrowseSort } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { CONTROL } from "./BrowseStates";

const SORTS: Record<BrowseSort, string> = {
  recent: "Newest",
  downloads: "Most downloaded · 30 days",
  views: "Most viewed · 30 days",
};

/** SortMenu orders the listing by first publication or by the last 30 days of downloads or views. */
export function SortMenu({
  className,
  filters,
  navigate,
}: {
  className?: string;
  filters: BrowseFilters;
  navigate: (next: BrowseFilters) => void;
}) {
  const sort = filters.sort ?? "recent";

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger className={cn(CONTROL, className)}>
        <ArrowDownWideNarrow
          aria-hidden="true"
          className="hidden size-4 text-mute sm:block"
        />
        <span className="sr-only">Sort: </span>
        <span className="truncate">{SORTS[sort]}</span>
        <ChevronDown aria-hidden="true" className="size-4 text-mute" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup
          onValueChange={(value) =>
            navigate({
              ...filters,
              sort: value === "recent" ? undefined : (value as BrowseSort),
            })
          }
          value={sort}
        >
          {Object.entries(SORTS).map(([value, label]) => (
            <DropdownMenuRadioItem
              className="pointer-fine:min-h-9"
              key={value}
              value={value}
            >
              {label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
