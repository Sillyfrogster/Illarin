"use client";

import { Check, Copy, X } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { cn, focusRing } from "@/lib/cn";

const SETTLE_MS = 2000;

const WORDS = { idle: "Copy", copied: "Copied", failed: "Failed" } as const;
const ICONS = { idle: Copy, copied: Check, failed: X } as const;

/** InputCopy is a filled field whose whole row copies the value it shows, the action word saying what happened. */
export function InputCopy({
  value,
  label,
  className,
}: {
  value: string;
  label?: string;
  className?: string;
}) {
  const [status, setStatus] = useState<keyof typeof WORDS>("idle");
  const labelId = useId();

  useEffect(() => {
    if (status === "idle") return;
    const timer = setTimeout(() => setStatus("idle"), SETTLE_MS);
    return () => clearTimeout(timer);
  }, [status]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setStatus("copied");
    } catch {
      setStatus("failed");
    }
  }

  const Icon = ICONS[status];
  return (
    <div className={cn("flex min-w-0 flex-col gap-1", className)}>
      {label ? (
        <span className="font-ui text-meta text-mute" id={labelId}>
          {label}
        </span>
      ) : null}
      <button
        aria-describedby={label ? labelId : undefined}
        className={cn(
          "group flex min-h-control w-full min-w-0 cursor-pointer items-center gap-3 rounded-control bg-fill px-3 text-left transition-colors duration-80 hover:bg-fill-hover",
          focusRing,
        )}
        onClick={copy}
        type="button"
      >
        <span className="min-w-0 flex-1 py-2 font-mono text-meta break-all text-ink">
          {value}
        </span>
        <span
          aria-live="polite"
          className={cn(
            "flex shrink-0 items-center gap-1.5 font-ui text-meta font-medium",
            status === "failed"
              ? "text-stop"
              : status === "copied"
                ? "text-accent"
                : "text-mute group-hover:text-ink",
          )}
        >
          <Icon aria-hidden="true" className="size-3.5" />
          {WORDS[status]}
        </span>
      </button>
    </div>
  );
}
