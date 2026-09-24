"use client";

import { motion, useReducedMotion } from "framer-motion";
import { FileArchive, FileImage, FileJson } from "lucide-react";
import { useId, useRef } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";

const SHEETS = [
  { Icon: FileImage, name: ".png", rest: -11, fanned: -20, shift: -44 },
  { Icon: FileJson, name: ".json", rest: 0, fanned: 0, shift: 0 },
  { Icon: FileArchive, name: ".charx", rest: 11, fanned: 20, shift: 44 },
] as const;

/** FannedSheets draws three file sheets that spread apart while a file is held over the page. */
export function FannedSheets({
  open,
  className,
}: {
  open: boolean;
  className?: string;
}) {
  const still = useReducedMotion();
  return (
    <div
      aria-hidden="true"
      className={cn("relative h-40 w-64 shrink-0", className)}
    >
      {SHEETS.map(({ Icon, name, rest, fanned, shift }, index) => (
        <motion.div
          animate={
            still
              ? { rotate: rest }
              : {
                  rotate: open ? fanned : rest,
                  x: open ? shift * 1.6 : shift,
                  y: open ? (index === 1 ? -16 : -2) : index === 1 ? -6 : 4,
                }
          }
          className={cn(
            "absolute top-2 left-1/2 -ml-[3.25rem] flex h-32 w-[6.5rem] origin-bottom flex-col justify-between rounded-control p-2.5 shadow-[0_14px_30px_-16px_rgb(0_0_0/0.45)]",
            index === 1 ? "z-10 bg-plane" : "bg-plane/90",
          )}
          initial={false}
          key={name}
          transition={{ type: "spring", stiffness: 320, damping: 22 }}
        >
          <Icon
            className={cn(index === 1 ? "text-accent" : "text-mute")}
            size={20}
            strokeWidth={1.6}
          />
          <span className="grid gap-1">
            <span className="h-1 w-4/5 rounded-full bg-rule" />
            <span className="h-1 w-3/5 rounded-full bg-rule" />
            <span className="mt-1 font-ui text-label font-medium text-mute">
              {name}
            </span>
          </span>
        </motion.div>
      ))}
    </div>
  );
}

/** DropStage is where a file is chosen; the page above it takes a file dropped anywhere. */
export function DropStage({
  onFile,
  over,
}: {
  onFile: (file: File) => void;
  over: boolean;
}) {
  const field = useId();
  const input = useRef<HTMLInputElement>(null);

  return (
    <section
      aria-labelledby={`${field}-heading`}
      className={cn(
        "group relative flex flex-col items-center gap-8 overflow-hidden rounded-plate px-6 py-10 transition-colors duration-300 sm:flex-row sm:gap-10 sm:px-10 sm:py-12",
        over ? "bg-accent-wash" : "bg-inset",
      )}
    >
      <FannedSheets
        className="transition-transform duration-500 ease-[var(--ease-wipe)] group-hover:-translate-y-1 motion-reduce:transform-none"
        open={over}
      />
      <div className="flex min-w-0 flex-col items-center text-center sm:items-start sm:text-left">
        <h2
          className="font-display text-title font-medium text-ink"
          id={`${field}-heading`}
        >
          {over ? "Let go to upload" : "Drop a file anywhere"}
        </h2>
        <p className="mt-2 max-w-[40ch] text-ui text-mute">
          A character card, lorebook, preset, theme, pack or extension zip. It
          uploads the moment you choose it.
        </p>
        <input
          className="sr-only"
          id={field}
          onChange={(event) => {
            const chosen = event.target.files?.[0];
            event.target.value = "";
            if (chosen) onFile(chosen);
          }}
          ref={input}
          tabIndex={-1}
          type="file"
        />
        <Button
          className="mt-6"
          onClick={() => input.current?.click()}
          variant="primary"
        >
          Choose a file
        </Button>
      </div>
    </section>
  );
}
