import Link from "next/link";
import { DefaultCover } from "@/components/media/DefaultCover";
import type { BrowseType, BrowseWork } from "@/lib/api/query";
import { buildBrowseHref } from "@/lib/browse-url";
import { cn } from "@/lib/cn";
import { TYPE_PLURALS } from "@/lib/work-types";

export interface TypeTile {
  type: BrowseType;
  count: number;
  covers: BrowseWork[];
}

/** FAN places the three covers and spreads them when the tile is pointed at. */
const FAN = [
  "-translate-x-[42%] -rotate-8 group-hover/type:-translate-x-[56%] group-hover/type:-rotate-12",
  "translate-x-[42%] rotate-8 group-hover/type:translate-x-[56%] group-hover/type:rotate-12",
  "-translate-y-1 group-hover/type:-translate-y-3",
];

/** TypeTiles links to each type on Browse with its live count and three of its covers fanned out. */
export function TypeTiles({ tiles }: { tiles: TypeTile[] }) {
  return (
    <ul className="m-0 grid list-none grid-cols-2 gap-4 p-0 sm:grid-cols-3 lg:grid-cols-5">
      {tiles.map((tile) => (
        <li key={tile.type}>
          <Link
            className="group/type flex h-full flex-col overflow-hidden rounded-card bg-inset p-5 text-ink ring-1 ring-ink/6 transition-colors duration-80 hover:bg-plane"
            href={buildBrowseHref({ type: tile.type })}
          >
            <div
              className="relative grid h-36 place-items-center"
              aria-hidden="true"
            >
              {FAN.map((place, index) => {
                const work = tile.covers[index];
                return (
                  <div
                    className={cn(
                      "absolute aspect-3/4 h-full overflow-hidden rounded-art bg-plane bg-cover bg-top shadow-cover ring-1 ring-ink/8 transition-transform duration-300 ease-(--ease-wipe) motion-reduce:transition-none",
                      place,
                    )}
                    key={place}
                    style={
                      work?.cover
                        ? { backgroundImage: `url(${work.cover.url})` }
                        : undefined
                    }
                  >
                    {work?.cover ? null : (
                      <DefaultCover compact type={tile.type} />
                    )}
                  </div>
                );
              })}
            </div>
            <p className="mt-5 text-ui font-medium">
              {TYPE_PLURALS[tile.type]}
            </p>
            <p className="text-meta text-mute tabular-nums">
              {tile.count.toLocaleString("en-US")}
            </p>
          </Link>
        </li>
      ))}
    </ul>
  );
}
