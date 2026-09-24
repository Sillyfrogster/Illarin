"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { AlertCircle, Check, FileText } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { TypeMark } from "@/components/browse/TypeMark";
import { Button } from "@/components/ui/button";
import type { BrowseType } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { fileWeight } from "@/lib/file-weight";
import { TYPE_LABELS } from "@/lib/work-types";

export type Reading =
  | { at: "sending"; file: File; sent: number }
  | { at: "reading"; file: File }
  | { at: "lost"; file: File }
  | { at: "found"; file: File; name: string; type: BrowseType }
  | { at: "refused"; file: File; message: string };

const EASE = [0.16, 1, 0.3, 1] as const;

const LINES = [92, 78, 85, 60, 88, 72, 40, 81, 66, 90, 74, 52, 86, 70];

/** ReadingFile shows the chosen file as a sheet that fills while it uploads, is scanned while Illarin reads it, and turns into the work it became. */
export function ReadingFile({
  reading,
  onBeginAgain,
  onCheckAgain,
}: {
  reading: Reading;
  onBeginAgain: () => void;
  onCheckAgain: () => void;
}) {
  const heading = useRef<HTMLHeadingElement>(null);
  const refused = reading.at === "refused";

  useEffect(() => {
    if (refused) heading.current?.focus();
  }, [refused]);

  const title = {
    sending: "Uploading",
    reading: "Reading your file",
    lost: "Import status unavailable",
    found: "Opening your draft",
    refused: "This file was not added",
  }[reading.at];

  return (
    <section
      aria-live="polite"
      className="grid items-center gap-8 sm:grid-cols-[13rem_minmax(0,1fr)] sm:gap-12"
    >
      <Sheet reading={reading} />

      <div className="min-w-0">
        <p className="truncate text-meta text-mute">
          {reading.file.name} · {fileWeight(reading.file.size)}
        </p>
        <h2
          className="mt-2 font-display text-title font-medium text-ink outline-none"
          ref={heading}
          tabIndex={-1}
        >
          {title}
          {reading.at === "sending" ? (
            <span className="ml-3 text-mute tabular-nums">
              {Math.round(reading.sent * 100)}%
            </span>
          ) : null}
        </h2>

        <AnimatePresence mode="wait">
          {reading.at === "found" ? (
            <motion.p
              animate={{ opacity: 1, y: 0 }}
              className="mt-4 flex items-center gap-2 text-lede text-ink wrap-anywhere"
              initial={{ opacity: 0, y: 8 }}
              key="found"
              transition={{ duration: 0.4, ease: EASE }}
            >
              <TypeMark
                className="size-5 shrink-0 text-accent"
                type={reading.type}
              />
              {reading.name}
              <span className="text-mute">
                · {TYPE_LABELS[reading.type] ?? reading.type}
              </span>
            </motion.p>
          ) : (
            <motion.p
              animate={{ opacity: 1 }}
              className={cn(
                "mt-3 max-w-[52ch] text-ui",
                refused ? "text-ink" : "text-mute",
              )}
              exit={{ opacity: 0 }}
              initial={{ opacity: 0 }}
              key={reading.at}
            >
              {
                {
                  sending: "Keep this page open until the upload finishes.",
                  reading: "You can leave this page. Illarin keeps reading.",
                  lost: "The connection dropped. Your file is safe. Check again to see where it got to.",
                  refused: reading.at === "refused" ? reading.message : "",
                }[reading.at]
              }
            </motion.p>
          )}
        </AnimatePresence>

        <Steps at={reading.at} />

        {refused ? (
          <Button className="mt-6" onClick={onBeginAgain} variant="primary">
            Choose another file
          </Button>
        ) : reading.at === "lost" ? (
          <Button className="mt-6" onClick={onCheckAgain} variant="primary">
            Check again
          </Button>
        ) : null}
      </div>
    </section>
  );
}

function Steps({ at }: { at: Reading["at"] }) {
  if (at === "refused") return null;
  const step = { sending: 0, reading: 1, lost: 1, found: 2 }[at];
  return (
    <ol className="mt-6 flex list-none items-center gap-2 p-0 text-meta">
      {["Upload", "Read", "Open"].map((name, index) => (
        <li className="flex items-center gap-2" key={name}>
          {index > 0 ? (
            <span
              aria-hidden="true"
              className="relative h-px w-6 overflow-hidden bg-rule sm:w-10"
            >
              <span
                className={cn(
                  "absolute inset-0 origin-left bg-action transition-transform duration-500 ease-[var(--ease-wipe)]",
                  index <= step ? "scale-x-100" : "scale-x-0",
                )}
              />
            </span>
          ) : null}
          <span
            className={cn(
              "flex items-center gap-1.5 transition-colors duration-300",
              index <= step ? "text-ink" : "text-mute",
            )}
          >
            {index < step || at === "found" ? (
              <Check
                aria-hidden="true"
                className="size-3.5 text-accent"
                strokeWidth={3}
              />
            ) : null}
            {name}
          </span>
        </li>
      ))}
    </ol>
  );
}

function Sheet({ reading }: { reading: Reading }) {
  const still = useReducedMotion();
  const picture = usePicture(reading.file);
  const scanning = reading.at === "reading" && !still;
  const refused = reading.at === "refused";
  const found = reading.at === "found";
  const filled = reading.at === "sending" ? reading.sent : 1;

  return (
    <motion.div
      animate={{
        opacity: 1,
        rotate: refused ? -3 : 0,
        y: 0,
        scale: found ? 1.03 : 1,
      }}
      className="relative mx-auto w-44 sm:w-full"
      initial={still ? false : { opacity: 0, rotate: -8, y: -36, scale: 0.94 }}
      transition={{ type: "spring", stiffness: 260, damping: 22 }}
    >
      <div
        className={cn(
          "relative aspect-[5/6] overflow-hidden rounded-plate shadow-[0_20px_45px_-20px_rgb(0_0_0/0.5)] transition-[filter] duration-500",
          picture ? "bg-inset" : "bg-plane",
          refused && "grayscale",
        )}
      >
        {picture ? (
          // biome-ignore lint/performance/noImgElement: a local object URL, not a site image
          <img alt="" className="size-full object-contain" src={picture} />
        ) : (
          <div className="flex h-full flex-col gap-2.5 p-5">
            <FileText
              aria-hidden="true"
              className="mb-2 text-accent"
              size={22}
              strokeWidth={1.6}
            />
            {LINES.map((width, index) => (
              <motion.span
                animate={
                  scanning
                    ? { scaleX: [0, 1, 1], opacity: [0.3, 1, 1] }
                    : { scaleX: 1, opacity: 1 }
                }
                className="block h-1.5 origin-left rounded-full bg-rule"
                // biome-ignore lint/suspicious/noArrayIndexKey: fixed decorative lines
                key={index}
                style={{ width: `${width}%` }}
                transition={
                  scanning
                    ? {
                        duration: 2.4,
                        times: [
                          index / LINES.length,
                          (index + 1) / LINES.length,
                          1,
                        ],
                        repeat: Number.POSITIVE_INFINITY,
                        ease: "easeOut",
                      }
                    : { duration: 0.3 }
                }
              />
            ))}
          </div>
        )}

        <div
          aria-hidden="true"
          className={cn(
            "absolute inset-0 origin-bottom bg-action/25 transition-[transform,opacity] duration-300 ease-out",
            reading.at === "sending" ? "opacity-100" : "opacity-0",
          )}
          style={{ transform: `scaleY(${filled})` }}
        />

        {scanning ? (
          <motion.div
            animate={{ top: ["-12%", "100%"] }}
            aria-hidden="true"
            className="absolute inset-x-0 h-[12%] bg-gradient-to-b from-transparent via-action/35 to-action/0"
            transition={{
              duration: 2.4,
              repeat: Number.POSITIVE_INFINITY,
              ease: "linear",
            }}
          >
            <span className="absolute inset-x-0 bottom-0 h-0.5 bg-action" />
          </motion.div>
        ) : null}
      </div>

      <AnimatePresence>
        {found || refused ? (
          <motion.span
            animate={{ scale: 1, opacity: 1 }}
            className={cn(
              "absolute -right-3 -bottom-3 flex size-11 items-center justify-center rounded-full shadow-[0_8px_20px_-8px_rgb(0_0_0/0.5)]",
              found ? "bg-action text-on-accent" : "bg-stop text-on-stop",
            )}
            initial={still ? false : { scale: 0, opacity: 0 }}
            transition={{
              type: "spring",
              stiffness: 500,
              damping: 20,
              delay: 0.1,
            }}
          >
            {found ? (
              <Check aria-hidden="true" size={20} strokeWidth={2.6} />
            ) : (
              <AlertCircle aria-hidden="true" size={20} strokeWidth={2.2} />
            )}
          </motion.span>
        ) : null}
      </AnimatePresence>
    </motion.div>
  );
}

/** usePicture shows a chosen image file as itself, since a card's picture is the card. */
function usePicture(file: File): string | null {
  const [address, setAddress] = useState<string | null>(null);
  useEffect(() => {
    if (!file.type.startsWith("image/")) return;
    const made = URL.createObjectURL(file);
    setAddress(made);
    return () => URL.revokeObjectURL(made);
  }, [file]);
  return address;
}
