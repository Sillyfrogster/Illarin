import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { TILE, TileArt, TileText } from "@/components/browse/WorkTile";
import { Scroller } from "@/components/ui/scroller";
import { buildBrowseHref } from "@/lib/browse-url";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_PLURALS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import type { HomeRow } from "./rows";

/** SLOT sizes a work so five show beside the plate on a wide screen, three on a tablet and two on a phone. */
const SLOT =
  "flex w-[44vw] shrink-0 snap-start sm:w-[calc((100%-2*1rem)/3)] lg:w-[calc((100%-4*1rem)/5)]";

/** TypeRow is one type's most downloaded works, led by that type's painted plate, which opens the full list on Browse. */
export function TypeRow({ row }: { row: HomeRow }) {
  const plural = TYPE_PLURALS[row.type];
  const heading = `home-${row.type}`;
  return (
    <section
      aria-labelledby={heading}
      className="-mb-5 flex gap-4 max-sm:flex-col max-sm:gap-1"
    >
      <Plate
        heading={heading}
        href={buildBrowseHref({ type: row.type, sort: "downloads" })}
        plural={plural}
        type={row.type}
      />
      <div className="min-w-0 flex-1">
        <Scroller className="flex snap-x snap-mandatory gap-4 pt-3 pb-8">
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
    </section>
  );
}

function Plate({
  heading,
  href,
  plural,
  type,
}: {
  heading: string;
  href: string;
  plural: string;
  type: string;
}) {
  return (
    <div className="group/plate relative isolate mt-3 mb-8 flex shrink-0 flex-col justify-end overflow-hidden rounded-card bg-inset p-5 max-sm:my-0 max-sm:h-28 max-sm:p-4 sm:w-[calc((100%-3*1rem)/4)] lg:w-[calc((100%-5*1rem)/6)] text-on-media ring-1 ring-ink/8 focus-within:ring-accent [html[data-artwork=off]_&]:text-ink">
      <div
        aria-hidden="true"
        className="absolute inset-0 -z-10 bg-cover bg-top max-sm:bg-[position:50%_18%] transition-transform duration-300 ease-(--ease-wipe) group-hover/plate:scale-[1.03] motion-reduce:transform-none"
        data-artwork
        style={{ backgroundImage: `url(/home/plates/${type}.webp)` }}
      >
        <div className="absolute inset-0 bg-linear-to-t from-media/85 via-media/30 to-transparent" />
      </div>
      <h2 className="font-display text-name-s font-medium" id={heading}>
        Popular {plural.toLowerCase()}
      </h2>
      <Link
        className="mt-1 flex items-center gap-1.5 text-meta text-inherit outline-none after:absolute after:inset-0 after:content-['']"
        href={href}
      >
        All {plural.toLowerCase()}
        <ArrowRight aria-hidden="true" className="size-3.5" />
      </Link>
    </div>
  );
}
