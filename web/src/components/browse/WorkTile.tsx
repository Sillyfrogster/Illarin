"use client";

import { EyeOff } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import { DefaultCover } from "@/components/media/DefaultCover";
import type { BrowseWork, NsfwPreference } from "@/lib/api/query";
import { tagSearchHref } from "@/lib/tag-search";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_LABELS } from "@/lib/work-types";

const TILE_TAGS = 4;

/** TileArt is a work's cover cropped to the card's 3:4 frame, its type's default cover when it has none, and a quiet mark when the cover is blurred. */
export function TileArt({
  eager,
  preference,
  work,
}: {
  eager: boolean;
  preference: NsfwPreference;
  work: BrowseWork;
}) {
  const [failed, setFailed] = useState(false);
  const blurred = work.isNsfw === true && preference !== "shown";

  return (
    <div
      className="relative aspect-3/4 overflow-hidden rounded-art bg-inset"
      data-tile-art={work.id}
    >
      {work.cover && !failed ? (
        <Image
          alt=""
          className="size-full object-cover object-top transition-transform duration-300 ease-(--ease-wipe) group-hover/tile:scale-[1.03] motion-reduce:transform-none"
          fill
          loading={eager ? "eager" : "lazy"}
          onError={() => setFailed(true)}
          sizes="(max-width: 639px) 46vw, (max-width: 1023px) 30vw, (max-width: 1439px) 23vw, 260px"
          src={work.cover.url}
          unoptimized
        />
      ) : (
        <DefaultCover compact type={work.type} />
      )}
      {blurred ? (
        <span className="absolute inset-x-0 bottom-0 flex items-center gap-1.5 bg-linear-to-t from-media/70 to-transparent px-3 pt-8 pb-2.5 text-meta font-medium text-on-media">
          <EyeOff aria-hidden="true" className="size-3.5" />
          Adult
        </span>
      ) : null}
    </div>
  );
}

/** TileText is the name, the type and creator, and a line of tags under a work's art. */
export function TileText({ work }: { work: BrowseWork }) {
  const name = workDisplayName(work.name);
  return (
    <div className="flex min-w-0 flex-col gap-1 px-1.5 pt-3 pb-1">
      <h3 className="line-clamp-2 text-ui leading-snug font-medium text-ink [overflow-wrap:anywhere]">
        {name}
      </h3>
      <p className="flex min-w-0 items-center gap-1.5 text-meta text-mute">
        <span className="shrink-0">{TYPE_LABELS[work.type]}</span>
        <span aria-hidden="true">·</span>
        <Link
          className="relative z-1 truncate hover:text-ink hover:underline"
          href={`/@${work.creator}`}
          prefetch={false}
        >
          @{work.creator}
        </Link>
      </p>
      {work.tags.length ? (
        <ul className="m-0 flex h-[1.5em] list-none flex-wrap gap-x-2 overflow-hidden p-0 text-meta text-mute">
          {work.tags.slice(0, TILE_TAGS).map((tag) => (
            <li className="max-w-full truncate" key={tag.value}>
              <Link
                className="relative z-1 hover:text-accent"
                href={tagSearchHref(tag.value)}
                prefetch={false}
              >
                #{tag.label}
              </Link>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
