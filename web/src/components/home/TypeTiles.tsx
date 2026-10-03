import { ArrowUpRight } from "lucide-react";
import Link from "next/link";
import { DefaultCover } from "@/components/media/DefaultCover";
import type { BrowseType, BrowseWork } from "@/lib/api/query";
import { buildBrowseHref } from "@/lib/browse-url";
import { TYPE_PLURALS } from "@/lib/work-types";

/** FAN is where the two covers behind the front one sit, at rest and when the tile is hovered. */
const FAN = [
  "rotate-[-4deg] -translate-x-[6%] group-hover/type:rotate-[-9deg] group-hover/type:-translate-x-[16%]",
  "rotate-[4deg] translate-x-[6%] group-hover/type:rotate-[9deg] group-hover/type:translate-x-[16%]",
];

export interface TypeTile {
  type: BrowseType;
  count: number;
  works: BrowseWork[];
}

/** TypeTiles is one tile per type, its three most downloaded works stacked and fanning out on hover, linking to Browse. */
export function TypeTiles({ tiles }: { tiles: TypeTile[] }) {
  return (
    <ul className="m-0 grid list-none grid-cols-2 gap-x-4 gap-y-8 p-0 sm:grid-cols-3 lg:grid-cols-5">
      {tiles.map(({ type, count, works }) => {
        const [front, ...behind] = works.slice(0, 3);
        return (
          <li className="min-w-0" key={type}>
            <Link
              className="group/type flex flex-col gap-4 text-ink"
              href={buildBrowseHref({ type })}
            >
              <div className="relative mx-[12%] aspect-3/4">
                {behind.map((work, at) => (
                  <Cover
                    className={`absolute inset-0 opacity-70 brightness-75 ${FAN[at]}`}
                    key={work.id}
                    type={type}
                    work={work}
                  />
                ))}
                <Cover
                  className="absolute inset-0 shadow-cover group-hover/type:-translate-y-1.5"
                  type={type}
                  work={front}
                />
              </div>
              <div className="flex items-baseline justify-between gap-3 border-t border-rule pt-3">
                <span className="truncate text-lede font-medium">
                  {TYPE_PLURALS[type]}
                </span>
                <span className="flex shrink-0 items-center gap-1 text-meta text-mute tabular-nums">
                  {count.toLocaleString("en-US")}
                  <ArrowUpRight
                    aria-hidden="true"
                    className="size-3.5 transition-transform duration-300 ease-(--ease-wipe) group-hover/type:translate-x-0.5 group-hover/type:-translate-y-0.5 group-hover/type:text-accent"
                  />
                </span>
              </div>
            </Link>
          </li>
        );
      })}
    </ul>
  );
}

/** Cover is one work's cover in the stack, or its type's default cover when it has none. */
function Cover({
  className,
  type,
  work,
}: {
  className: string;
  type: BrowseType;
  work?: BrowseWork;
}) {
  return (
    <div
      className={`overflow-hidden rounded-art bg-inset ring-1 ring-ink/10 transition-[translate,rotate] duration-500 ease-(--ease-wipe) ${className}`}
    >
      {work?.cover ? (
        // biome-ignore lint/performance/noImgElement: small fixed covers, same as the rail
        <img
          alt=""
          className="size-full object-cover object-top"
          decoding="async"
          loading="lazy"
          src={work.cover.url}
        />
      ) : (
        <DefaultCover compact type={type} />
      )}
    </div>
  );
}
