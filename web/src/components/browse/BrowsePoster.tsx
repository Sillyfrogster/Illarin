"use client";

import { motion, useReducedMotion } from "framer-motion";
import { CircleHelp, EyeOff } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { type ReactNode, useState } from "react";
import type { BrowseWork, NsfwPreference } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { posterFace, type TypeSetting, typeSetting } from "@/lib/poster-face";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import { TypeMark } from "./TypeMark";

const SETTING: Record<TypeSetting, string> = {
  grand: "text-[clamp(1.9rem,2.9vw,2.6rem)] leading-[1.02]",
  large: "text-[clamp(1.5rem,2.1vw,1.95rem)] leading-[1.07]",
  medium: "text-[clamp(1.15rem,1.5vw,1.4rem)] leading-[1.15]",
  small: "text-[clamp(0.95rem,1.1vw,1.05rem)] leading-[1.35]",
};

const PLATE =
  "overflow-hidden rounded-plate transition duration-500 group-hover:-translate-y-1 motion-reduce:transform-none motion-reduce:transition-none";

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
          transition: { type: "spring", stiffness: 300, damping: 30 } as const,
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
        <div className={cn(PLATE, "relative aspect-5/6 bg-inset")}>
          {work.cover ? (
            <Image
              alt=""
              className="size-full object-contain transition-transform duration-700 group-hover:scale-[1.02] motion-reduce:transform-none motion-reduce:transition-none"
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
            "flex aspect-5/6 min-w-0 flex-col justify-between p-5",
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
              "min-w-0 font-display font-medium tracking-[-0.035em] text-balance [overflow-wrap:anywhere] transition-colors duration-300 motion-reduce:transition-none",
              groundFor(work.id).title,
              SETTING[typeSetting(name)],
            )}
          >
            {name}
          </p>
        </div>
      )}

      <div className="pt-3.5">
        <h3 className="line-clamp-2 font-display text-[1.0625rem] leading-[1.3] font-medium tracking-[-0.015em] [overflow-wrap:anywhere] transition-colors duration-200 group-hover:text-accent motion-reduce:transition-none">
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
                className="relative z-1 -my-3 inline-flex min-h-11 min-w-0 items-center [overflow-wrap:anywhere] hover:text-ink hover:underline"
                href={`/@${work.creator}`}
                prefetch={false}
              >
                @{work.creator}
              </Link>
            </>
          ) : null}
        </p>

        {apps?.length ? (
          <p className="mt-1.5 font-ui text-label text-mute">
            <span className="sr-only">Works in </span>
            {apps.join(" · ")}
          </p>
        ) : null}

        <div className="mt-2.5 flex flex-wrap gap-2 empty:hidden">
          {work.ownerState ? (
            <span className="inline-flex min-h-6 items-center rounded-control bg-accent-wash px-2 font-ui text-label font-medium text-accent capitalize">
              {work.ownerState}
            </span>
          ) : null}
          {work.isNsfw === null ? (
            <span className="inline-flex min-h-6 items-center gap-1.5 rounded-control bg-deep px-2 font-ui text-label text-mute">
              <CircleHelp aria-hidden="true" className="size-3" />
              Rating not set
            </span>
          ) : null}
          {work.isNsfw ? (
            <span className="inline-flex min-h-6 items-center gap-1.5 rounded-control bg-stop-wash px-2 font-ui text-label font-medium text-stop">
              {blurred ? (
                <EyeOff aria-hidden="true" className="size-3" />
              ) : null}
              {blurred ? "Adult · blurred" : "Adult"}
            </span>
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
          {new Date(takedown.at).toLocaleString("en-GB", {
            dateStyle: "medium",
            timeStyle: "short",
          })}
        </time>
      </p>
    </div>
  );
}
