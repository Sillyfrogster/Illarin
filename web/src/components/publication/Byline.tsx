"use client";

import Image from "next/image";
import Link from "next/link";
import type { PostByline } from "@/lib/api/query";
import { archivePath } from "@/lib/blog-paths";
import { bylineName, bylineProfilePath } from "@/lib/byline";
import { useOrigins } from "@/lib/origins";

export function Byline({ byline }: { byline: PostByline }) {
  const { site } = useOrigins();
  const name = bylineName(byline);
  const path = bylineProfilePath(byline);
  const profile = path ? new URL(path, site).href : null;
  return (
    <div className="flex min-w-0 items-center gap-3">
      <span
        aria-hidden="true"
        className="grid size-11 shrink-0 place-items-center overflow-hidden rounded-full bg-deep"
      >
        {byline.avatar ? (
          <Image
            alt=""
            className="size-full object-cover"
            height={44}
            src={byline.avatar.url}
            unoptimized
            width={44}
          />
        ) : (
          <span className="font-display text-[19px] leading-none text-mute">
            {byline.handle.slice(0, 1).toUpperCase()}
          </span>
        )}
      </span>
      <span className="flex min-w-0 flex-col">
        {profile ? (
          <a
            className="truncate text-ui font-medium text-ink hover:text-accent"
            href={profile}
          >
            {name}
          </a>
        ) : (
          <span className="truncate text-ui font-medium text-ink">{name}</span>
        )}
        <span className="flex min-w-0 flex-wrap items-baseline gap-x-2 text-meta text-mute max-sm:whitespace-normal">
          {byline.app ? (
            <>
              <Link
                className="truncate text-ink hover:text-accent"
                href={archivePath("app", byline.app.slug)}
              >
                {byline.app.name}
              </Link>
              <span className="truncate">Published by Illarin</span>
            </>
          ) : (
            <>
              <span className="truncate text-ink">Illarin Team</span>
              {standingOf(byline).length > 0 ? (
                <span className="min-w-0 truncate">
                  {standingOf(byline).join(" · ")}
                </span>
              ) : null}
            </>
          )}
        </span>
      </span>
    </div>
  );
}

export function BylineText({
  affiliation,
  byline,
}: {
  affiliation: boolean;
  byline: PostByline;
}) {
  return (
    <span className="inline-flex min-w-0 flex-wrap items-baseline gap-x-2">
      <span className="font-medium">{bylineName(byline)}</span>
      {affiliation ? <span>{byline.app?.name ?? "Illarin Team"}</span> : null}
    </span>
  );
}

function standingOf(byline: PostByline): string[] {
  return [...byline.positions, ...byline.distinctions];
}
