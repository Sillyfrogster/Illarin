"use client";

import { CircleHelp, EyeOff } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import type { BrowseAsset, NsfwVisibility } from "@/lib/api/query";
import { assetDisplayName } from "@/lib/asset-name";
import { assetHref } from "@/lib/asset-url";
import { cn } from "@/lib/cn";
import { KIND_LABELS } from "@/lib/kinds";
import { posterFace, type TypeSetting, typeSetting } from "@/lib/poster-face";
import { KindMark } from "./KindMark";

/** How large the name is set when the plate has to carry it alone. */
const SETTING: Record<TypeSetting, string> = {
  grand: "text-[clamp(1.9rem,2.9vw,2.6rem)] leading-[1.02]",
  large: "text-[clamp(1.5rem,2.1vw,1.95rem)] leading-[1.07]",
  medium: "text-[clamp(1.15rem,1.5vw,1.4rem)] leading-[1.15]",
  small: "text-[clamp(0.95rem,1.1vw,1.05rem)] leading-[1.35]",
};

const PLATE =
  "overflow-hidden rounded-plate transition duration-500 group-hover:-translate-y-1 motion-reduce:transform-none motion-reduce:transition-none";

/** Three plate grounds, so a wall of titled plates has rhythm rather than one flat colour. */
const GROUNDS = [
  {
    plate: "bg-media dark:bg-plane",
    title: "text-on-media group-hover:text-over-mute",
  },
  { plate: "bg-accent-wash", title: "text-ink group-hover:text-accent" },
  { plate: "bg-deep", title: "text-ink group-hover:text-accent" },
];

/** Spread over the grounds by the asset's own id, the way a handle picks an avatar tone. */
function groundFor(id: string) {
  let hash = 0;
  for (const character of id) {
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) % 100000;
  }
  return GROUNDS[hash % GROUNDS.length];
}

/**
 * One result in the catalog. A creator who gave their work a picture gets a
 * picture; a creator who did not gets their title set across the whole plate,
 * because a catalog of prompts and worlds is mostly words, and a work's own
 * name says more about it than an empty frame does.
 */
export function CatalogPoster({
  asset,
  eager = false,
  visibility,
}: {
  asset: BrowseAsset;
  eager?: boolean;
  visibility: NsfwVisibility;
}) {
  const [failed, setFailed] = useState(false);
  const face = posterFace({ cover: asset.cover, failed });
  const name = assetDisplayName(asset.name);
  const blurred = asset.isNsfw === true && visibility !== "shown";

  const title = (
    <Link
      className="[color:inherit] after:absolute after:inset-0 after:content-['']"
      href={assetHref(asset.id, asset.name)}
    >
      {name}
    </Link>
  );

  return (
    <li className="group relative flex min-w-0 flex-col">
      {face === "art" ? (
        <div className={cn(PLATE, "relative aspect-5/6 bg-media")}>
          {asset.cover ? (
            <Image
              alt=""
              className="size-full object-contain transition-transform duration-700 group-hover:scale-[1.02] motion-reduce:transform-none motion-reduce:transition-none"
              fill
              loading={eager ? "eager" : "lazy"}
              onError={() => setFailed(true)}
              sizes="(max-width: 639px) 46vw, (max-width: 1023px) 30vw, (max-width: 1439px) 23vw, 250px"
              src={asset.cover.url}
              unoptimized
            />
          ) : null}
        </div>
      ) : (
        <div
          className={cn(
            PLATE,
            "grid aspect-5/6 grid-cols-[minmax(0,1fr)] content-end",
            groundFor(asset.id).plate,
          )}
        >
          <h3
            className={cn(
              "min-w-0 p-5 font-display font-medium tracking-[-0.035em] text-balance [overflow-wrap:anywhere] transition-colors duration-300 motion-reduce:transition-none",
              groundFor(asset.id).title,
              SETTING[typeSetting(name)],
            )}
          >
            {title}
          </h3>
        </div>
      )}

      <div className="pt-4">
        {face === "art" ? (
          <h3 className="font-display text-[clamp(1.05rem,1.25vw,1.2rem)] leading-[1.25] font-medium tracking-[-0.02em] [overflow-wrap:anywhere] transition-colors duration-200 group-hover:text-accent motion-reduce:transition-none">
            {title}
          </h3>
        ) : null}

        <p
          className={cn(
            "flex flex-wrap items-center gap-x-2 gap-y-1 font-ui text-meta text-mute",
            face === "art" && "mt-2",
          )}
        >
          <KindMark
            className="size-3.5 shrink-0 text-accent"
            kind={asset.kind}
          />
          {KIND_LABELS[asset.kind]}
          <span aria-hidden="true">·</span>
          <Link
            className="relative z-1 py-3 [overflow-wrap:anywhere] hover:text-ink hover:underline"
            href={`/@${asset.creator}`}
          >
            @{asset.creator}
          </Link>
        </p>

        <div className="mt-2.5 flex flex-wrap gap-2 empty:hidden">
          {asset.ownerState ? (
            <span className="inline-flex min-h-6 items-center rounded-control bg-accent-wash px-2 font-ui text-label font-medium text-accent capitalize">
              {asset.ownerState}
            </span>
          ) : null}
          {asset.isNsfw === null ? (
            <span className="inline-flex min-h-6 items-center gap-1.5 rounded-control bg-deep px-2 font-ui text-label text-mute">
              <CircleHelp aria-hidden="true" className="size-3" />
              Rating not set
            </span>
          ) : null}
          {asset.isNsfw ? (
            <span className="inline-flex min-h-6 items-center gap-1.5 rounded-control bg-stop-wash px-2 font-ui text-label font-medium text-stop">
              {blurred ? (
                <EyeOff aria-hidden="true" className="size-3" />
              ) : null}
              {blurred ? "Adult · blurred" : "Adult"}
            </span>
          ) : null}
        </div>

        {asset.withhold ? <Withheld withhold={asset.withhold} /> : null}
      </div>
    </li>
  );
}

function Withheld({
  withhold,
}: {
  withhold: NonNullable<BrowseAsset["withhold"]>;
}) {
  return (
    <div className="relative z-1 mt-3 rounded-control bg-stop-wash p-3">
      <p className="font-ui text-meta font-medium text-stop">
        {withhold.reason}
      </p>
      <p className="mt-1 font-ui text-label text-mute">
        @{withhold.actor} ·{" "}
        <time dateTime={withhold.at}>
          {new Date(withhold.at).toLocaleString("en-GB", {
            dateStyle: "medium",
            timeStyle: "short",
          })}
        </time>
      </p>
    </div>
  );
}
