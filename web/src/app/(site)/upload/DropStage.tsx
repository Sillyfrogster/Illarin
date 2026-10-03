"use client";

import { Upload } from "lucide-react";
import { useId, useRef } from "react";
import { TypeMark } from "@/components/browse/TypeMark";
import { Button } from "@/components/ui/button";
import type { BrowseType } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { TYPE_LABELS } from "@/lib/work-types";

const READS: [BrowseType, string][] = [
  ["character", "Image, CHARX or JSON card"],
  ["lorebook", "JSON"],
  ["preset", "SillyTavern or Lumiverse JSON"],
  ["theme", "SillyTavern JSON or Lumiverse zip"],
  ["pack", "Lumiverse JSON"],
  ["extension", "SillyTavern or Spindle zip"],
];

/** DropStage is where a file is chosen, with the formats Illarin reads for each type; the page around it takes a file dropped anywhere. */
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
      className="flex min-w-0 flex-col gap-2 rounded-card bg-inset p-2"
    >
      <div
        className={cn(
          "flex flex-1 flex-col items-center justify-center rounded-plate border border-dashed px-6 py-8 text-center sm:min-h-72 sm:py-10 transition-colors duration-160",
          over ? "border-accent bg-accent-wash" : "border-rule bg-field",
        )}
      >
        <span className="flex size-12 items-center justify-center rounded-plate bg-accent-wash text-accent">
          <Upload aria-hidden="true" className="size-5" strokeWidth={1.8} />
        </span>
        <h2
          className="mt-5 font-display text-section font-medium text-ink"
          id={`${field}-heading`}
        >
          Upload a file
        </h2>
        <p className="mt-1.5 max-w-[34ch] text-ui text-mute">
          <span className="pointer-coarse:hidden">
            Drop it anywhere on this page, or choose one.{" "}
          </span>
          It uploads as soon as you pick it.
        </p>
        <input
          hidden
          onChange={(event) => {
            const chosen = event.target.files?.[0];
            event.target.value = "";
            if (chosen) onFile(chosen);
          }}
          ref={input}
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

      <div className="px-4 pt-3 pb-2">
        <h3 className="text-meta font-medium text-ink">Illarin reads</h3>
        <ul className="mt-2 grid list-none gap-x-6 p-0 sm:grid-cols-2">
          {READS.map(([type, formats]) => (
            <li
              className="flex min-h-9 items-center gap-2.5 border-b border-rule/60 text-meta last:border-b-0 sm:nth-last-2:border-b-0"
              key={type}
            >
              <TypeMark className="size-4 shrink-0 text-mute" type={type} />
              <span className="text-ink">{TYPE_LABELS[type]}</span>
              <span className="ml-auto truncate text-mute">{formats}</span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
