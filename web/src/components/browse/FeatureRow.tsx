"use client";

import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import type { BrowseFilters, BrowsePage } from "@/lib/api/query";
import { chooseFilter } from "@/lib/browse-url";

type Facet = BrowsePage["facets"][number];
type Option = Facet["options"][number];

const FEATURE =
  "inline-flex h-control w-auto items-center gap-2 border border-edge px-3 font-ui text-ui font-medium text-ink hover:border-accent/50 aria-pressed:border-accent/50 [&_.count]:text-meta [&_.count]:font-normal [&_.count]:text-mute [&_.count]:tabular-nums aria-pressed:[&_.count]:text-accent";

// Yes or no features offer "Included", which reads as the feature's own name.
function isPresence(facet: Facet) {
  return facet.options.some((option) => option.value === "true");
}

// A "None" answer stays visible only when an old link still asks for it.
function shown(facet: Facet) {
  return isPresence(facet)
    ? facet.options.filter(
        (option) => option.value === "true" || option.selected,
      )
    : facet.options;
}

function nameOf(facet: Facet, option: Option) {
  return isPresence(facet) && option.value === "true"
    ? facet.label
    : isPresence(facet)
      ? `${facet.label}: ${option.label}`
      : option.label;
}

/** FeatureRow shows the features a type offers as toggles with the number of works each matches. */
export function FeatureRow({
  facets,
  filters,
  navigate,
}: {
  facets: Facet[];
  filters: BrowseFilters;
  navigate: (next: BrowseFilters) => void;
}) {
  if (!facets.length) return null;
  const inUse = facets.filter((facet) =>
    facet.options.some((option) => option.selected),
  ).length;

  return (
    <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
      {facets.map((facet) => (
        <fieldset
          aria-label={facet.label}
          className="m-0 flex min-w-0 flex-wrap items-center gap-2 border-0 p-0"
          key={facet.key}
        >
          {isPresence(facet) ? null : (
            <span className="font-ui text-meta text-mute">{facet.label}</span>
          )}
          {shown(facet).map((option) => (
            <Toggle
              className={FEATURE}
              disabled={option.count === 0 && !option.selected}
              key={option.value}
              onPressedChange={(pressed) =>
                navigate(
                  chooseFilter(
                    filters,
                    facet.key,
                    pressed ? option.value : null,
                  ),
                )
              }
              pressed={option.selected}
            >
              {nameOf(facet, option)}
              <span className="count border-l border-edge pl-2">
                {option.count}
              </span>
            </Toggle>
          ))}
        </fieldset>
      ))}
      {inUse > 1 ? (
        <Button
          onClick={() => navigate({ ...filters, facet: undefined })}
          size="compact"
          variant="ghost"
        >
          Clear filters
        </Button>
      ) : null}
    </div>
  );
}
