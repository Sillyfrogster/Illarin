"use client";

import { motion, type PanInfo } from "framer-motion";
import { FileText, ImageIcon, SlidersHorizontal } from "lucide-react";
import {
  type CSSProperties,
  type PointerEvent,
  type ReactNode,
  type RefObject,
  useEffect,
  useRef,
} from "react";
import type { RecordedVersion } from "@/lib/api/query";
import { cn, focusRing } from "@/lib/cn";
import { useMediaQuery } from "@/lib/use-media-query";
import { versionTag } from "@/lib/version-label";
import { versionTitle } from "@/lib/work-versions";
import { listedVersions, useCardStage } from "./stage";

const SWIPE_DISTANCE = 70;
const SHUFFLE_MS = 640;

/** cardNameSize steps a long name down so it wraps in a few lines on the card instead of being cut. */
export function cardNameSize(name: string): string {
  const length = Array.from(name).length;
  if (length > 48) return "text-name-s";
  if (length > 24) return "text-name-m";
  return "text-name-l";
}

function shortDate(version: RecordedVersion): string {
  return new Date(version.recordedAt).toLocaleDateString("en-US", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

/** versionName is a version's creator label when it has one, else its place in the history. */
export function versionName(version: RecordedVersion): string {
  return version.versionLabel
    ? versionTag(version.versionLabel)
    : versionTitle(version);
}

/** WorkCard is the work as one card: its cover and name on the front, what its file holds on the back, and an older version rising behind it when the reader points at one. */
export function WorkCard({
  art,
  back,
  backLabel,
  name,
  details,
  hasBack,
  live,
}: {
  art: ReactNode;
  back?: ReactNode;
  backLabel?: string;
  name: ReactNode;
  details: ReactNode;
  hasBack: boolean;
  live: boolean;
}) {
  const stage = useCardStage();
  const coarse = useMediaQuery("(pointer: coarse)");
  const tilt = useRef<HTMLDivElement>(null);
  const versions = listedVersions(stage.versions);
  const swipes = live && coarse && versions.length > 1 && !stage.turned;
  const shown = stage.viewing ?? versions[0] ?? null;

  useShuffle(stage.card, stage.viewing?.number ?? null);

  const { turn, turned } = stage;
  useEffect(() => {
    if (turned && !hasBack) turn(false);
  }, [hasBack, turn, turned]);

  function lean(event: PointerEvent<HTMLDivElement>) {
    if (!live || event.pointerType !== "mouse" || !tilt.current) return;
    const box = event.currentTarget.getBoundingClientRect();
    const x = (event.clientX - box.left) / box.width;
    const y = (event.clientY - box.top) / box.height;
    const style = tilt.current.style;
    style.setProperty("--mx", `${Math.round(x * 100)}%`);
    style.setProperty("--my", `${Math.round(y * 100)}%`);
    style.setProperty("--rx", `${((0.5 - y) * 5).toFixed(2)}deg`);
    style.setProperty("--ry", `${((x - 0.5) * 6).toFixed(2)}deg`);
  }

  function settle() {
    const style = tilt.current?.style;
    style?.setProperty("--rx", "0deg");
    style?.setProperty("--ry", "0deg");
  }

  function swiped(_: unknown, info: PanInfo) {
    const at = shown
      ? versions.findIndex((one) => one.number === shown.number)
      : 0;
    if (info.offset.x < -SWIPE_DISTANCE && at < versions.length - 1) {
      stage.view(versions[at + 1] ?? null);
    } else if (info.offset.x > SWIPE_DISTANCE && at > 0) {
      stage.view(at - 1 === 0 ? null : (versions[at - 1] ?? null));
    }
  }

  return (
    <div className="relative isolate">
      {live ? <Peek version={stage.peek} /> : null}
      <motion.div
        drag={swipes ? "x" : false}
        dragElastic={0.35}
        dragSnapToOrigin
        onDragEnd={swiped}
        className={swipes ? "touch-pan-y" : undefined}
      >
        <div
          className="group/card [perspective:1400px]"
          onPointerLeave={settle}
          onPointerMove={lean}
        >
          <div
            className="relative [transform:rotateX(var(--rx,0deg))_rotateY(var(--ry,0deg))] transition-transform duration-300 ease-(--ease-wipe)"
            ref={tilt}
            style={{ "--mx": "50%", "--my": "30%" } as CSSProperties}
          >
            <div ref={stage.card}>
              <div
                className={cn(
                  "relative grid transition-transform duration-[560ms] ease-[cubic-bezier(0.22,1,0.36,1)] [transform-style:preserve-3d]",
                  stage.turned && "[transform:rotateY(180deg)]",
                )}
              >
                <div
                  aria-hidden={stage.turned || undefined}
                  className="relative flex flex-col overflow-hidden rounded-card bg-plane p-3 shadow-card ring-1 ring-ink/8 [grid-area:1/1] [backface-visibility:hidden]"
                  inert={stage.turned || undefined}
                >
                  <div className="relative" data-card-art>
                    {art}
                  </div>
                  <div className="flex flex-col gap-1.5 px-2 pt-4 pb-2">
                    {stage.viewing ? (
                      <p className="text-meta font-medium text-accent">
                        Viewing {versionName(stage.viewing)} ·{" "}
                        {shortDate(stage.viewing)}
                      </p>
                    ) : null}
                    {name}
                    <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
                      {details}
                      {hasBack ? <TurnButton label={backLabel} /> : null}
                    </div>
                  </div>
                  {live ? (
                    <span
                      aria-hidden="true"
                      className="pointer-events-none absolute inset-0 rounded-card bg-[radial-gradient(circle_at_var(--mx)_var(--my),rgb(255_255_255/0.14),transparent_55%)] opacity-0 mix-blend-soft-light transition-opacity duration-300 group-hover/card:opacity-100"
                    />
                  ) : null}
                </div>
                {hasBack ? (
                  <div
                    aria-hidden={!stage.turned || undefined}
                    className="absolute inset-0 flex flex-col rounded-card bg-plane shadow-card ring-1 ring-ink/8 [backface-visibility:hidden] [transform:rotateY(180deg)]"
                    inert={!stage.turned || undefined}
                  >
                    <div
                      className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 pt-5 pb-3"
                      ref={stage.setBackSlot}
                    />
                    <div className="flex justify-end border-t border-rule/60 px-3 py-2">
                      <TurnButton label={backLabel} />
                    </div>
                    {back}
                  </div>
                ) : null}
              </div>
            </div>
            <Delivery />
          </div>
        </div>
      </motion.div>
    </div>
  );
}

/** useShuffle slides the card out and back when the reader moves to another version, so the change reads as a different card. */
function useShuffle(
  card: RefObject<HTMLDivElement | null>,
  number: number | null,
) {
  const shown = useRef(number);
  useEffect(() => {
    if (shown.current === number) return;
    shown.current = number;
    if (!card.current) return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    card.current.animate(
      [
        { transform: "none" },
        {
          transform: "translate(-16%, -2%) rotate(-6deg) scale(0.97)",
          offset: 0.42,
        },
        {
          transform: "translate(-4%, 1%) rotate(-1.5deg) scale(0.99)",
          offset: 0.7,
        },
        { transform: "none" },
      ],
      { duration: SHUFFLE_MS, easing: "cubic-bezier(0.22, 1, 0.36, 1)" },
    );
  }, [card, number]);
}

/** Peek pulls the version the reader points at on the timeline up from behind the card, so they see which card they would bring forward. */
function Peek({ version }: { version: RecordedVersion | null }) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "absolute inset-0 -z-1 overflow-hidden rounded-card bg-fill ring-1 ring-ink/8 transition-[transform,opacity] duration-300 ease-(--ease-wipe)",
        version
          ? "[transform:translateY(-64px)_rotate(-1.5deg)] opacity-100"
          : "opacity-0",
      )}
    >
      {version ? (
        <p className="truncate px-5 pt-3.5 text-meta text-ink">
          <span className="font-medium">{versionName(version)}</span>
          <span className="text-mute"> · {shortDate(version)}</span>
          {version.summary ? (
            <span className="text-mute"> · {version.summary}</span>
          ) : null}
        </p>
      ) : null}
    </div>
  );
}

function TurnButton({ label }: { label?: string }) {
  const stage = useCardStage();
  const Icon = stage.turned ? ImageIcon : label ? SlidersHorizontal : FileText;
  return (
    <button
      aria-pressed={stage.turned}
      className={cn(
        "-mr-2 inline-flex h-control-compact items-center gap-1.5 rounded-control px-2 text-meta font-medium text-ink transition-colors duration-150 ease-(--ease-wipe) hover:bg-fill hover:text-accent [&_svg]:size-3.5",
        focusRing,
      )}
      onClick={() => stage.turn(!stage.turned)}
      type="button"
    >
      <Icon aria-hidden="true" />
      {stage.turned ? "Show the cover" : (label ?? "What's in the file")}
    </button>
  );
}

/** Delivery traces the card's edge while a connected app has yet to collect a send. */
function Delivery() {
  const { collecting } = useCardStage();
  if (!collecting) return null;
  return (
    <svg
      aria-hidden="true"
      className="pointer-events-none absolute -inset-[3px] size-[calc(100%+6px)] overflow-visible"
    >
      <rect
        className="animate-trace fill-none stroke-accent"
        height="100%"
        pathLength={100}
        rx="23"
        strokeDasharray="22 78"
        strokeLinecap="round"
        strokeWidth="1.5"
        width="100%"
      />
    </svg>
  );
}
