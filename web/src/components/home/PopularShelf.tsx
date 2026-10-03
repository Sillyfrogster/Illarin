"use client";

import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { TILE, TileArt, TileText } from "@/components/browse/WorkTile";
import { Scroller } from "@/components/ui/scroller";
import { Segmented } from "@/components/ui/segmented";
import type { BrowseType } from "@/lib/api/query";
import { buildBrowseHref } from "@/lib/browse-url";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_PLURALS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import type { HomeRow } from "./rows";

/** SLOT sizes a card so five fit a wide row, three a tablet row and two a phone row. */
export const SLOT =
  "flex w-[44vw] shrink-0 snap-start sm:w-[calc((100%-2*1rem)/3)] lg:w-[calc((100%-4*1rem)/5)]";

/** PopularShelf shows one type's most downloaded works at a time, with a switch between the types that have enough of them. */
export function PopularShelf({ rows }: { rows: HomeRow[] }) {
  const [type, setType] = useState<BrowseType>(rows[0].type);
  const row = rows.find((one) => one.type === type) ?? rows[0];
  const plural = TYPE_PLURALS[row.type].toLowerCase();

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-4">
        {rows.length > 1 ? (
          <Segmented
            aria-label="Type"
            onValueChange={setType}
            options={rows.map((one) => ({
              value: one.type,
              label: TYPE_PLURALS[one.type],
            }))}
            value={type}
          />
        ) : null}
        <Link
          className="flex min-h-control items-center gap-1.5 text-ui font-medium"
          href={buildBrowseHref({ type: row.type, sort: "downloads" })}
        >
          All {plural}
          <ArrowRight aria-hidden="true" className="size-4" />
        </Link>
      </div>
      <Scroller className="mt-4 flex snap-x snap-mandatory gap-4 pt-1 pb-8">
        {row.page.items.map((work, index) => (
          <div className={SLOT} key={work.id}>
            <div className={TILE}>
              <TileArt
                eager={index < 5}
                preference={row.page.nsfwPreference}
                work={work}
              />
              <TileText work={work} />
              <Link
                aria-label={workDisplayName(work.name)}
                className="absolute inset-0 rounded-card outline-none"
                href={workHref(work.id, work.name)}
                prefetch={false}
              />
            </div>
          </div>
        ))}
      </Scroller>
    </div>
  );
}
