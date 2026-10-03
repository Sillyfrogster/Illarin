"use client";

import { Download } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { TileArt } from "@/components/browse/WorkTile";
import type { BrowseWork, NsfwPreference } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import "./home.css";

const APPS = ["SillyTavern", "RisuAI", "Lumiverse"];
const TURN_MS = 2600;

/** AppDemo shows one real work whose download button follows the reader's app, with a butterfly flying to the app it is for. */
export function AppDemo({
  preference,
  work,
}: {
  preference: NsfwPreference;
  work: BrowseWork;
}) {
  const [at, setAt] = useState(0);

  useEffect(() => {
    const turn = setInterval(
      () => setAt((now) => (now + 1) % APPS.length),
      TURN_MS,
    );
    return () => clearInterval(turn);
  }, []);

  return (
    <div className="rounded-card bg-inset p-6 sm:p-8">
      <div className="flex items-center gap-4">
        <div className="w-20 shrink-0">
          <TileArt eager={false} preference={preference} work={work} />
        </div>
        <div className="min-w-0">
          <p className="truncate text-ui font-medium text-ink">
            {workDisplayName(work.name)}
          </p>
          <p className="text-meta text-mute">
            {TYPE_LABELS[work.type]} · @{work.creator}
          </p>
        </div>
      </div>
      <Link
        className="mt-6 flex h-control items-center justify-center gap-2 rounded-control bg-action px-4 text-ui font-medium text-on-accent hover:bg-action-hover"
        href={workHref(work.id, work.name)}
      >
        <Download aria-hidden="true" className="size-4" />
        <span aria-live="off" className="tabular-nums">
          Download for {APPS[at]}
        </span>
      </Link>
      <div className="relative mt-10 grid grid-cols-3 gap-2" aria-hidden="true">
        <span
          data-artwork
          className="home-butterfly-pin absolute -top-8 h-6 w-9 -translate-x-1/2 transition-[left] duration-700 ease-(--ease-wipe) motion-reduce:transition-none"
          style={{ left: `${(at * 2 + 1) * (100 / (APPS.length * 2))}%` }}
        >
          <i />
          <i />
        </span>
        {APPS.map((app, index) => (
          <span
            className={cn(
              "flex h-control items-center justify-center rounded-control text-meta font-medium transition-colors duration-300",
              index === at ? "bg-accent-wash text-accent" : "bg-fill text-mute",
            )}
            key={app}
          >
            {app}
          </span>
        ))}
      </div>
    </div>
  );
}
