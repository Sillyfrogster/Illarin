"use client";

import { motion, type PanInfo, useReducedMotion } from "framer-motion";
import { ArrowRight, ChevronLeft, ChevronRight, X } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useRef, ViewTransition } from "react";
import { DefaultCover } from "@/components/media/DefaultCover";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import type { BrowseWork, NsfwPreference } from "@/lib/api/query";
import { tagSearchHref } from "@/lib/tag-search";
import { holdScroll, workCoverTransition } from "@/lib/view-transitions";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import { GetIt, Holds, heldFormat, usePeek } from "./peek";

const SWIPE = 60;
const GLOW_W = 6;
const GLOW_H = 4;

/** Glow paints the cover into a few pixels and stretches them under the panel, so the panel takes on the work's colours without a live blur. */
function Glow({ src }: { src: string }) {
  const canvas = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const image = new window.Image();
    image.onload = () => {
      const context = canvas.current?.getContext("2d");
      context?.drawImage(image, 0, 0, GLOW_W, GLOW_H);
    };
    image.src = src;
    return () => {
      image.onload = null;
    };
  }, [src]);
  return (
    <canvas
      className="pointer-events-none absolute inset-0 -z-1 size-full rounded-card opacity-30 dark:opacity-40"
      height={GLOW_H}
      ref={canvas}
      width={GLOW_W}
    />
  );
}

/** OpenPanel is a work opened inside Browse's grid: its cover, blurb and file, the button that gets it, and the way to its page. */
export function OpenPanel({
  id,
  next,
  onClose,
  preference,
  previous,
  work,
}: {
  id: string;
  next: (() => void) | null;
  onClose: () => void;
  preference: NsfwPreference;
  previous: (() => void) | null;
  work: BrowseWork;
}) {
  const still = useReducedMotion();
  const box = useRef<HTMLLIElement>(null);
  const peek = usePeek(work.id, true);
  const format = peek.work ? heldFormat(peek.work) : null;
  const blurred = work.isNsfw === true && preference !== "shown";
  const href = workHref(work.id, work.name);

  function swiped(_: unknown, info: PanInfo) {
    if (info.offset.x < -SWIPE) next?.();
    if (info.offset.x > SWIPE) previous?.();
  }

  return (
    <motion.li
      animate={{ opacity: 1 }}
      className="relative z-10 col-span-full scroll-mt-[calc(var(--site-header-offset)+4.5rem)]"
      exit={{ opacity: 0, transition: { duration: 0.12 } }}
      id={id}
      initial={{ opacity: 0 }}
      onAnimationComplete={() =>
        box.current?.scrollIntoView({
          block: "nearest",
          behavior: still ? "auto" : "smooth",
        })
      }
      ref={box}
      transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
    >
      <motion.div
        className="relative isolate mt-3 mb-2 touch-pan-y rounded-card bg-plane p-3 shadow-card ring-1 ring-ink/8 sm:p-4"
        drag={next || previous ? "x" : false}
        dragElastic={0.25}
        dragSnapToOrigin
        onDragEnd={swiped}
      >
        {work.cover ? <Glow key={work.id} src={work.cover.url} /> : null}

        <div className="grid gap-5 md:grid-cols-[minmax(0,20rem)_minmax(0,1fr)] lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)_minmax(0,18rem)] lg:gap-8">
          <ViewTransition {...workCoverTransition(work.id)}>
            <div
              className="relative z-2 mx-auto aspect-3/4 w-full max-w-80 overflow-hidden rounded-art bg-inset shadow-cover md:mx-0 md:max-w-none"
              key={work.id}
            >
              {work.cover ? (
                <Image
                  alt=""
                  className="object-cover object-top"
                  fill
                  sizes="(max-width: 767px) 80vw, 22rem"
                  src={work.cover.url}
                  unoptimized
                />
              ) : (
                <DefaultCover type={work.type} />
              )}
              {blurred ? (
                <span className="absolute inset-x-0 bottom-0 bg-linear-to-t from-media/70 to-transparent px-4 pt-10 pb-3 text-meta font-medium text-on-media">
                  Adult
                </span>
              ) : null}
            </div>
          </ViewTransition>

          <div className="flex min-w-0 flex-col gap-4 md:py-2">
            <div className="flex flex-col gap-1.5 pr-28">
              <h3 className="text-name-m font-medium tracking-[-0.02em] text-balance text-ink [overflow-wrap:anywhere]">
                <Link
                  className="[color:inherit] hover:text-accent"
                  href={href}
                  onClick={holdScroll}
                  transitionTypes={["nav-forward"]}
                >
                  {workDisplayName(work.name)}
                </Link>
              </h3>
              <p className="text-meta text-mute">
                {TYPE_LABELS[work.type]} ·{" "}
                <Link
                  className="hover:text-ink hover:underline"
                  href={`/@${work.creator}`}
                >
                  @{work.creator}
                </Link>
              </p>
            </div>
            {peek.work ? (
              peek.work.blurb ? (
                <p className="line-clamp-6 max-w-[62ch] text-prose text-ink">
                  {peek.work.blurb}
                </p>
              ) : null
            ) : peek.failed ? (
              <p className="text-meta text-mute">
                Illarin could not read this work. Open its page instead.
              </p>
            ) : (
              <div className="flex max-w-[62ch] flex-col gap-2">
                <Skeleton className="h-3.5 w-full" />
                <Skeleton className="h-3.5 w-11/12" />
                <Skeleton className="h-3.5 w-2/3" />
              </div>
            )}
            {work.tags.length ? (
              <ul className="m-0 flex list-none flex-wrap gap-x-3 gap-y-1 p-0 text-meta">
                {work.tags.map((tag) => (
                  <li key={tag.value}>
                    <Link
                      className="text-mute hover:text-accent"
                      href={tagSearchHref(tag.value)}
                    >
                      #{tag.label}
                    </Link>
                  </li>
                ))}
              </ul>
            ) : null}
            <div className="mt-auto flex flex-col gap-3 lg:hidden">
              {peek.work ? (
                <GetIt
                  apps={peek.apps}
                  className="max-w-80"
                  key={peek.work.id}
                  work={peek.work}
                />
              ) : null}
            </div>
            <Link
              className="inline-flex items-center gap-1.5 self-start text-ui font-medium text-accent hover:underline"
              href={href}
              onClick={holdScroll}
              transitionTypes={["nav-forward"]}
            >
              Open the page
              <ArrowRight aria-hidden="true" className="size-4" />
            </Link>
          </div>

          <div className="flex min-w-0 flex-col gap-4 md:col-span-2 lg:col-span-1 lg:pt-12 lg:pb-2">
            <div className="hidden lg:block">
              {peek.work ? (
                <GetIt apps={peek.apps} key={peek.work.id} work={peek.work} />
              ) : (
                <Skeleton className="h-control w-full" />
              )}
            </div>
            {format ? (
              <div>
                <p className="text-meta text-mute">
                  In the {format.label} file
                </p>
                <Holds className="mt-1" format={format} />
              </div>
            ) : null}
          </div>
        </div>

        <div className="absolute top-3 right-3 flex gap-1 sm:top-4 sm:right-4">
          <Button
            aria-label="Previous work"
            disabled={!previous}
            onClick={() => previous?.()}
            size="icon"
            variant="ghost"
          >
            <ChevronLeft aria-hidden="true" />
          </Button>
          <Button
            aria-label="Next work"
            disabled={!next}
            onClick={() => next?.()}
            size="icon"
            variant="ghost"
          >
            <ChevronRight aria-hidden="true" />
          </Button>
          <Button
            aria-label="Close"
            onClick={onClose}
            size="icon"
            variant="ghost"
          >
            <X aria-hidden="true" />
          </Button>
        </div>
      </motion.div>
    </motion.li>
  );
}
