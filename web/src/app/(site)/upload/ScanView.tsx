"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import {
  AlertCircle,
  Check,
  File,
  FileImage,
  FileJson,
  ScrollText,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { TypeMark } from "@/components/browse/TypeMark";
import { Button } from "@/components/ui/button";
import type { BrowseType } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { fileWeight } from "@/lib/file-weight";
import { TYPE_LABELS } from "@/lib/work-types";
import type { ScanPart } from "./scan-file";

export type Upload =
  | { at: "sending"; sent: number }
  | { at: "reading" }
  | { at: "lost" }
  | { at: "found"; name: string; type: BrowseType }
  | { at: "refused"; message: string };

const EASE = [0.16, 1, 0.3, 1] as const;

/** The whole local scan takes about this long, however many parts the file holds. */
const SCAN_MS = 3600;

const KIND_ICONS = {
  picture: FileImage,
  card: ScrollText,
  data: FileJson,
  file: File,
} as const;

/** ScanView walks through the parts of a chosen file while it uploads, then shows what Illarin made of it. */
export function ScanView({
  file,
  parts,
  upload,
  onScanned,
  onBeginAgain,
  onCheckAgain,
}: {
  file: File;
  parts: ScanPart[] | null;
  upload: Upload;
  onScanned: () => void;
  onBeginAgain: () => void;
  onCheckAgain: () => void;
}) {
  const heading = useRef<HTMLHeadingElement>(null);
  const refused = upload.at === "refused";
  const total = parts?.length ?? 0;
  const hurry = upload.at === "found" || refused;
  const steady = Math.min(650, Math.max(110, SCAN_MS / Math.max(total, 1)));
  const pace = hurry ? Math.max(120, steady / 2) : steady;
  const scanned = useReveal(total, pace, onScanned);
  const scanning = parts !== null && scanned < total;
  const shown: Upload =
    scanning && upload.at === "found" ? { at: "reading" } : upload;

  useEffect(() => {
    if (refused) heading.current?.focus();
  }, [refused]);

  const title = scanning
    ? `Scanning ${scanned + 1} of ${total}`
    : {
        sending: "Uploading",
        reading: "Illarin is reading your file",
        lost: "Import status unavailable",
        found: "Opening your draft",
        refused: "This file was not added",
      }[upload.at];

  return (
    <section
      aria-live="polite"
      className="grid gap-8 md:grid-cols-[16rem_minmax(0,1fr)] md:gap-10"
    >
      <Viewer
        pace={pace}
        parts={parts ?? []}
        scanned={scanned}
        upload={shown}
      />

      <div className="min-w-0">
        <p className="truncate text-meta text-mute">
          {file.name} · {fileWeight(file.size)}
        </p>
        <h2
          className="mt-1.5 font-display text-title font-medium text-ink tabular-nums outline-none"
          ref={heading}
          tabIndex={-1}
        >
          {title}
        </h2>

        {shown.at === "found" ? (
          <motion.p
            animate={{ opacity: 1, y: 0 }}
            className="mt-2 flex items-center gap-2 text-lede text-ink wrap-anywhere"
            initial={{ opacity: 0, y: 6 }}
            transition={{ duration: 0.35, ease: EASE }}
          >
            <TypeMark
              className="size-5 shrink-0 text-accent"
              type={shown.type}
            />
            {shown.name}
            <span className="text-mute">· {TYPE_LABELS[shown.type]}</span>
          </motion.p>
        ) : refused ? (
          <p className="mt-2 max-w-[60ch] text-ui text-ink">{upload.message}</p>
        ) : upload.at === "lost" ? (
          <p className="mt-2 text-ui text-mute">
            The connection dropped. Your file is safe. Check again to see where
            it got to.
          </p>
        ) : null}

        {refused ? null : <Carrying upload={upload} />}

        <PartList parts={parts} refused={refused} scanned={scanned} />

        {refused ? (
          <Button className="mt-6" onClick={onBeginAgain} variant="primary">
            Choose another file
          </Button>
        ) : upload.at === "lost" ? (
          <Button className="mt-6" onClick={onCheckAgain} variant="primary">
            Check again
          </Button>
        ) : null}
      </div>
    </section>
  );
}

/** Carrying shows the file travelling to Illarin beside the scan: bytes sent, then Illarin reading. */
function Carrying({ upload }: { upload: Upload }) {
  const sent = upload.at === "sending" ? upload.sent : 1;
  const label =
    upload.at === "sending"
      ? `Sending to Illarin · ${Math.round(sent * 100)}%`
      : upload.at === "found"
        ? "Illarin read it"
        : "Sent · Illarin is reading it";
  return (
    <div className="mt-5">
      <div className="flex items-center justify-between text-meta text-mute tabular-nums">
        <span>{label}</span>
      </div>
      <div className="relative mt-2 h-1 overflow-hidden rounded-full bg-deep">
        <div
          className="absolute inset-0 origin-left rounded-full bg-action transition-transform duration-300 ease-out"
          style={{ transform: `scaleX(${sent})` }}
        />
        {upload.at === "reading" ? (
          <span className="absolute inset-y-0 w-1/3 bg-white/50 motion-safe:animate-sweep motion-reduce:hidden" />
        ) : null}
      </div>
    </div>
  );
}

function PartList({
  parts,
  scanned,
  refused,
}: {
  parts: ScanPart[] | null;
  scanned: number;
  refused: boolean;
}) {
  const list = useRef<HTMLOListElement>(null);
  const still = useReducedMotion();

  useEffect(() => {
    const node = list.current;
    if (node && scanned >= 0)
      node.scrollTo({
        top: node.scrollHeight,
        behavior: still ? "auto" : "smooth",
      });
  }, [scanned, still]);

  if (!parts) {
    return <p className="mt-6 text-meta text-mute">Opening the file…</p>;
  }
  const shown = parts.slice(0, scanned + 1);
  const pictures = parts.filter((part) => part.kind === "picture").length;

  return (
    <div className="mt-6">
      <p className="flex items-baseline justify-between gap-4 text-meta text-mute">
        <span className="font-medium text-ink">In this file</span>
        <span className="tabular-nums">
          {parts.length} {parts.length === 1 ? "part" : "parts"}
          {pictures > 0
            ? ` · ${pictures} ${pictures === 1 ? "picture" : "pictures"}`
            : ""}
        </span>
      </p>
      <ol
        className="mt-2 grid max-h-[19rem] list-none gap-1 overflow-y-auto overscroll-contain p-0 pr-1 [scrollbar-color:var(--v-rule)_transparent] [scrollbar-width:thin]"
        ref={list}
      >
        <AnimatePresence initial={false}>
          {shown.map((part, index) => {
            const current = index === scanned && !refused;
            const Icon = KIND_ICONS[part.kind];
            return (
              <motion.li
                animate={{ opacity: 1, x: 0 }}
                className={cn(
                  "flex min-h-12 items-center gap-3 rounded-control px-2 py-1.5 transition-colors duration-300",
                  current ? "bg-accent-wash" : "bg-transparent",
                )}
                initial={still ? false : { opacity: 0, x: -10 }}
                key={part.id}
                transition={{ duration: 0.25, ease: EASE }}
              >
                <span className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-[6px] bg-deep">
                  {part.picture ? (
                    // biome-ignore lint/performance/noImgElement: a picture read from the chosen file
                    <img
                      alt=""
                      className="size-full object-cover"
                      src={part.picture}
                    />
                  ) : (
                    <Icon
                      aria-hidden="true"
                      className="size-4 text-mute"
                      strokeWidth={1.6}
                    />
                  )}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-mono text-[0.75rem] text-ink">
                    {part.name}
                  </span>
                  <span className="block truncate text-label text-mute">
                    {part.note}
                  </span>
                </span>
                <span className="shrink-0 text-label text-mute tabular-nums">
                  {fileWeight(part.size)}
                </span>
                <span className="flex size-5 shrink-0 items-center justify-center">
                  {current ? (
                    <span className="size-1.5 rounded-full bg-accent motion-safe:animate-pulse" />
                  ) : (
                    <Check
                      aria-hidden="true"
                      className="size-4 text-accent"
                      strokeWidth={2.4}
                    />
                  )}
                </span>
              </motion.li>
            );
          })}
        </AnimatePresence>
      </ol>
    </div>
  );
}

function Viewer({
  parts,
  scanned,
  pace,
  upload,
}: {
  parts: ScanPart[];
  scanned: number;
  pace: number;
  upload: Upload;
}) {
  const still = useReducedMotion();
  const done = scanned >= parts.length;
  const pictures = parts.filter((part) => part.picture);
  const waiting = done && (upload.at === "sending" || upload.at === "reading");
  const cycled = useCycle(waiting ? pictures.length : 0, 1600);
  const cover =
    pictures.find((part) => /icon|avatar|main/i.test(part.name)) ?? pictures[0];

  const part: ScanPart | undefined = !done
    ? parts[scanned]
    : upload.at === "found" || upload.at === "refused"
      ? (cover ?? parts[0])
      : (pictures[cycled] ?? parts.at(-1));
  const sweeping =
    !still &&
    (!done || waiting) &&
    upload.at !== "refused" &&
    upload.at !== "found";

  return (
    <div className="relative mx-auto w-52 md:w-full">
      <div
        className={cn(
          "relative aspect-[5/6] overflow-hidden rounded-plate bg-media shadow-[0_20px_45px_-20px_rgb(0_0_0/0.5)] transition-[filter] duration-500",
          upload.at === "refused" && "grayscale",
        )}
      >
        <AnimatePresence initial={false} mode="popLayout">
          {part ? (
            <motion.div
              animate={{ opacity: 1, x: 0, filter: "blur(0px)" }}
              className="absolute inset-0"
              exit={{ opacity: 0, x: -24, filter: "blur(6px)" }}
              initial={
                still ? false : { opacity: 0, x: 24, filter: "blur(6px)" }
              }
              key={part.id}
              transition={{ duration: 0.28, ease: EASE }}
            >
              <PartFace part={part} pace={pace} still={Boolean(still)} />
            </motion.div>
          ) : null}
        </AnimatePresence>

        {sweeping && part ? (
          <motion.div
            animate={{ top: ["-14%", "100%"] }}
            aria-hidden="true"
            className="pointer-events-none absolute inset-x-0 h-[14%] bg-gradient-to-b from-transparent to-[color-mix(in_oklab,var(--v-action)_45%,transparent)]"
            key={`scan-${part.id}-${cycled}`}
            transition={{
              duration: (done ? 1600 : pace) / 1000,
              ease: "linear",
            }}
          >
            <span className="absolute inset-x-0 bottom-0 h-0.5 bg-[#b89aff] shadow-[0_0_12px_2px_rgb(124_58_237/0.7)]" />
          </motion.div>
        ) : null}

        {part ? (
          <p className="absolute inset-x-0 bottom-0 truncate bg-gradient-to-t from-black/75 to-transparent px-3 pt-8 pb-2.5 font-mono text-label text-on-media">
            {part.name}
          </p>
        ) : null}
      </div>

      <AnimatePresence>
        {upload.at === "found" || upload.at === "refused" ? (
          <motion.span
            animate={{ scale: 1, opacity: 1 }}
            className={cn(
              "absolute -right-3 -bottom-3 flex size-11 items-center justify-center rounded-full shadow-[0_8px_20px_-8px_rgb(0_0_0/0.5)]",
              upload.at === "found"
                ? "bg-action text-on-accent"
                : "bg-stop text-on-stop",
            )}
            initial={still ? false : { scale: 0, opacity: 0 }}
            transition={{
              type: "spring",
              stiffness: 500,
              damping: 20,
              delay: 0.1,
            }}
          >
            {upload.at === "found" ? (
              <Check aria-hidden="true" size={20} strokeWidth={2.6} />
            ) : (
              <AlertCircle aria-hidden="true" size={20} strokeWidth={2.2} />
            )}
          </motion.span>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

function PartFace({
  part,
  pace,
  still,
}: {
  part: ScanPart;
  pace: number;
  still: boolean;
}) {
  if (part.picture) {
    return (
      // biome-ignore lint/performance/noImgElement: a picture read from the chosen file
      <img alt="" className="size-full object-contain" src={part.picture} />
    );
  }
  if (part.lines) {
    const step = pace / 1000 / (part.lines.length + 1);
    return (
      <div className="flex h-full flex-col gap-1.5 overflow-hidden p-4 pb-10">
        <p className="mb-1 text-label font-medium text-[#b89aff]">
          {part.note}
        </p>
        {part.lines.map((line, index) => (
          <motion.p
            animate={{ opacity: 1, y: 0 }}
            className="font-mono text-[0.6875rem] leading-snug break-words text-on-media/85"
            initial={still ? false : { opacity: 0, y: 4 }}
            // biome-ignore lint/suspicious/noArrayIndexKey: lines of one fixed part
            key={index}
            transition={{ delay: index * step, duration: 0.2 }}
          >
            {line}
          </motion.p>
        ))}
      </div>
    );
  }
  const Icon = KIND_ICONS[part.kind];
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 p-4 text-on-media/80">
      <Icon aria-hidden="true" size={36} strokeWidth={1.3} />
      <p className="text-label">{part.note}</p>
    </div>
  );
}

/** useReveal counts through the parts one at a time, telling the caller once every part has shown. */
function useReveal(total: number, pace: number, onDone: () => void) {
  const [shown, setShown] = useState(0);
  const finish = useRef(onDone);
  finish.current = onDone;

  useEffect(() => {
    if (total === 0) return;
    if (shown >= total) {
      finish.current();
      return;
    }
    const timer = setTimeout(() => setShown((count) => count + 1), pace);
    return () => clearTimeout(timer);
  }, [shown, total, pace]);

  return shown;
}

function useCycle(length: number, every: number) {
  const [index, setIndex] = useState(0);
  useEffect(() => {
    if (length < 2) return;
    const timer = setInterval(() => setIndex((at) => (at + 1) % length), every);
    return () => clearInterval(timer);
  }, [length, every]);
  return length === 0 ? 0 : index % length;
}
