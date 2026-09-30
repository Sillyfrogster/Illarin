"use client";

import { motion, useReducedMotion } from "framer-motion";
import { CircleHelp, EyeOff } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { type ReactNode, useState } from "react";
import { Badge, BadgeLink } from "@/components/ui/badge";
import type { BrowseWork, NsfwPreference } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { posterFace, type TypeSetting, typeSetting } from "@/lib/poster-face";
import { spring } from "@/lib/springs";
import { tagSearchHref } from "@/lib/tag-search";
import { workCounts } from "@/lib/work-counts";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import { TypeMark } from "./TypeMark";

const CARD_TAGS = 3;

const SETTING: Record<TypeSetting, string> = {
  grand: "text-[clamp(1.25rem,17cqi,2.6rem)] leading-[1.02]",
  large: "text-[clamp(1.1rem,12.5cqi,1.95rem)] leading-[1.07]",
  medium: "text-[clamp(1rem,9cqi,1.4rem)] leading-[1.15]",
  small: "text-[clamp(0.85rem,6.5cqi,1.05rem)] leading-[1.35]",
};

const PLATE =
  "overflow-hidden rounded-plate transition duration-240 group-hover:-translate-y-1 motion-reduce:transform-none motion-reduce:transition-none";

const GROUNDS = [
  {
    plate: "bg-media dark:bg-plane",
    title: "text-on-media group-hover:text-over-mute",
  },
  { plate: "bg-accent-wash", title: "text-ink group-hover:text-accent" },
  { plate: "bg-deep", title: "text-ink group-hover:text-accent" },
];

function groundFor(id: string) {
  let hash = 0;
  for (const character of id) {
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) % 100000;
  }
  return GROUNDS[hash % GROUNDS.length];
}

export function BrowsePoster({
  action,
  animated = false,
  apps,
  byline = true,
  className,
  work,
  eager = false,
  preference,
}: {
  action?: ReactNode;
  animated?: boolean;
  apps?: string[];
  byline?: boolean;
  className?: string;
  work: BrowseWork;
  eager?: boolean;
  preference: NsfwPreference;
}) {
  const [failed, setFailed] = useState(false);
  const still = useReducedMotion();
  const face = posterFace({ cover: work.cover, failed });
  const name = workDisplayName(work.name);
  const blurred = work.isNsfw === true && preference !== "shown";

  const title = (
    <Link
      className="[color:inherit] after:absolute after:inset-0 after:content-['']"
      href={workHref(work.id, work.name)}
      prefetch={false}
    >
      {name}
    </Link>
  );

  const Card = animated ? motion.li : "li";
  const arrival =
    animated && !still
      ? {
          layout: true,
          initial: { opacity: 0, scale: 0.92 },
          animate: { opacity: 1, scale: 1 },
          exit: { opacity: 0, scale: 0.92 },
          transition: spring.slow,
        }
      : {};
  return (
    <Card
      className={cn(
        "group relative flex min-w-0 flex-col rounded-plate outline-offset-4 focus-within:outline-2 focus-within:outline-accent",
        className,
      )}
      {...arrival}
    >
      {action ? (
        <div className="absolute top-2.5 right-2.5 z-2">{action}</div>
      ) : null}
      {face === "art" ? (
        <div className={cn(PLATE, "relative aspect-3/4 bg-inset")}>
          {work.cover ? (
            <Image
              alt=""
              className="size-full object-cover object-top transition-transform duration-240 group-hover:scale-[1.02] motion-reduce:transform-none motion-reduce:transition-none"
              fill
              loading={eager ? "eager" : "lazy"}
              onError={() => setFailed(true)}
              sizes="(max-width: 639px) 46vw, (max-width: 1023px) 30vw, (max-width: 1439px) 23vw, 250px"
              src={work.cover.url}
              unoptimized
            />
          ) : null}
        </div>
      ) : (
        <div
          className={cn(
            PLATE,
            "@container flex aspect-3/4 min-w-0 flex-col justify-between p-5",
            groundFor(work.id).plate,
          )}
        >
          <TypeMark
            aria-hidden="true"
            className="size-6 text-accent"
            type={work.type}
          />
          <p
            aria-hidden="true"
            className={cn(
              "min-w-0 font-display font-medium tracking-[-0.035em] text-balance [overflow-wrap:anywhere] transition-colors duration-240 motion-reduce:transition-none",
              groundFor(work.id).title,
              SETTING[typeSetting(name)],
            )}
          >
            {name}
          </p>
        </div>
      )}

      <div className="pt-3.5">
        <h3 className="line-clamp-2 font-display text-[1.0625rem] leading-[1.3] font-medium tracking-[-0.015em] [overflow-wrap:anywhere] transition-colors duration-160 group-hover:text-accent motion-reduce:transition-none">
          {title}
        </h3>

        <p className="mt-1.5 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 font-ui text-meta text-mute">
          <TypeMark
            className="size-3.5 shrink-0 text-accent"
            type={work.type}
          />
          {TYPE_LABELS[work.type]}
          {byline ? (
            <>
              <span aria-hidden="true">·</span>
              <Link
                className="relative z-1 -my-3 inline-flex min-h-control min-w-0 items-center [overflow-wrap:anywhere] hover:text-ink hover:underline"
                href={`/@${work.creator}`}
                prefetch={false}
              >
                @{work.creator}
              </Link>
            </>
          ) : null}
        </p>

        <p className="mt-1 font-ui text-label text-mute tabular-nums">
          {workCounts(work.viewCount, work.downloadCount)}
        </p>

        {apps?.length ? (
          <p className="mt-1.5 font-ui text-label text-mute">
            <span className="sr-only">Works in </span>
            {apps.join(" · ")}
          </p>
        ) : null}

        {work.tags.length ? (
          <ul className="m-0 mt-2 flex list-none flex-wrap gap-1.5 p-0">
            {work.tags.slice(0, CARD_TAGS).map((tag) => (
              <li className="max-w-full" key={tag.value}>
                <BadgeLink
                  className="relative z-1"
                  href={tagSearchHref(tag.value)}
                  prefetch={false}
                >
                  {tag.label}
                </BadgeLink>
              </li>
            ))}
          </ul>
        ) : null}

        <div className="mt-2.5 flex flex-wrap gap-2 empty:hidden">
          {work.ownerState ? (
            <Badge className="capitalize" tone="accent">
              {work.ownerState}
            </Badge>
          ) : null}
          {work.isNsfw === null ? (
            <Badge>
              <CircleHelp aria-hidden="true" className="size-3" />
              Rating not set
            </Badge>
          ) : null}
          {work.isNsfw ? (
            <Badge tone="stop">
              {blurred ? (
                <EyeOff aria-hidden="true" className="size-3" />
              ) : null}
              {blurred ? "Adult · blurred" : "Adult"}
            </Badge>
          ) : null}
        </div>

        {work.takedown ? <TakenDown takedown={work.takedown} /> : null}
      </div>
    </Card>
  );
}

function TakenDown({
  takedown,
}: {
  takedown: NonNullable<BrowseWork["takedown"]>;
}) {
  return (
    <div className="relative z-1 mt-3 rounded-control bg-stop-wash p-3">
      <p className="font-ui text-meta font-medium text-stop">
        {takedown.reason}
      </p>
      <p className="mt-1 font-ui text-label text-mute">
        Illarin staff ·{" "}
        <time dateTime={takedown.at}>
          {new Date(takedown.at).toLocaleString("en-US", {
            dateStyle: "medium",
            timeStyle: "short",
          })}
        </time>
      </p>
    </div>
  );
}
