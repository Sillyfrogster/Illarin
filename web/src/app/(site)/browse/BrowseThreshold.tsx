"use client";

import { BrowseSearch } from "@/components/browse/BrowseSearch";
import { useBrowseNavigation } from "@/components/browse/use-browse-navigation";
import { Shell } from "@/components/layout/Shell";
import type { BrowseFilters } from "@/lib/api/query";

export function BrowseThreshold({ filters }: { filters: BrowseFilters }) {
  const { navigate } = useBrowseNavigation("/browse");

  return (
    <div className="bg-inset">
      <Shell className="grid items-center gap-6 pt-12 pb-8 md:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] md:gap-12 lg:pt-14 lg:pb-10">
        <div>
          <h1 className="font-display text-[clamp(2rem,3vw,2.75rem)] leading-[1.1] font-medium tracking-[-0.035em] text-balance">
            Browse
          </h1>
          <p className="mt-3 text-ui text-mute">
            Explore the community’s creations.
          </p>
        </div>
        <div className="min-w-0">
          <BrowseSearch
            hint="Search names and blurbs, or use tag:fantasy and author:handle."
            id="browse-search"
            label="Search works"
            onSearch={(q) => navigate({ ...filters, q })}
            placeholder="What are you looking for?"
            size="hero"
            value={filters.q ?? ""}
          />
        </div>
      </Shell>
    </div>
  );
}
