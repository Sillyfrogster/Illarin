"use client";

import Image from "next/image";
import { CatalogSearch } from "@/components/catalog/CatalogSearch";
import { useCatalogNavigation } from "@/components/catalog/use-catalog-navigation";
import { Shell } from "@/components/layout/Shell";
import type { BrowseFilters } from "@/lib/api/query";

const DISSOLVE = "linear-gradient(to bottom, #000 72%, transparent 99%)";

export function BrowseThreshold({ filters }: { filters: BrowseFilters }) {
  const { navigate } = useCatalogNavigation("/browse");

  return (
    <div className="relative isolate overflow-hidden">
      <Shell>
        <Image
          alt=""
          className="pointer-events-none relative -mr-[calc(var(--gutter)/2)] ml-auto block h-[172px] w-auto max-w-none object-contain object-bottom select-none sm:absolute sm:right-[-4%] sm:bottom-0 sm:-z-1 sm:mr-0 sm:h-[240px] md:h-[276px] lg:right-0 lg:h-[330px] xl:right-[2%] xl:h-[372px]"
          height={1254}
          priority
          sizes="(max-width: 767px) 276px, 372px"
          src="/landing/watcher-portrait.webp"
          style={{ maskImage: DISSOLVE, WebkitMaskImage: DISSOLVE }}
          width={1254}
        />
        <div className="pb-9 sm:max-w-[27rem] sm:pt-12 sm:pb-10 md:max-w-[31rem] lg:max-w-[40rem] lg:pt-16 lg:pb-14 xl:max-w-[44rem]">
          <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
            Browse the catalog
            <span className="text-accent">?</span>
          </h1>
          <div className="mt-6 max-w-[34rem] lg:mt-7">
            <CatalogSearch
              hint="Names, creators and blurbs. Narrow with tag:fantasy or author:handle."
              id="browse-search"
              label="Search the catalog"
              onSearch={(q) => navigate({ ...filters, q })}
              placeholder="Search the catalog"
              size="hero"
              value={filters.q ?? ""}
            />
          </div>
        </div>
      </Shell>
    </div>
  );
}
