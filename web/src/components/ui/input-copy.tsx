"use client";

import { AnimatePresence, motion } from "framer-motion";
import { Check, Copy, X } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { cn, focusRing } from "@/lib/cn";
import { spring } from "@/lib/springs";

const SETTLE_MS = 2000;

const WORDS = { idle: "Copy", copied: "Copied", failed: "Failed" } as const;
const ICONS = { idle: Copy, copied: Check, failed: X } as const;

/** InputCopy is Fluid Functionalism's copy field: the whole row copies the value it shows, and the action word swaps to say what happened. */
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
          "group flex min-h-control w-full min-w-0 cursor-pointer items-center gap-3 rounded-control border border-edge px-3 text-left transition-colors duration-80 hover:border-accent/50 focus-visible:border-accent",
          focusRing,
          "focus-visible:ring-0",
        )}
        onClick={copy}
        type="button"
      >
        <span className="min-w-0 flex-1 py-2 font-mono text-meta break-all text-ink">
          <mark className="rounded-[3px] bg-transparent text-ink transition-colors duration-80 group-hover:bg-accent-wash">
            {value}
          </mark>
        </span>
        <span
          aria-live="polite"
          className={cn(
            "inline-grid shrink-0 font-ui text-meta font-medium transition-colors duration-80",
            status === "failed"
              ? "text-stop"
              : "text-mute group-hover:text-ink",
          )}
        >
          <AnimatePresence initial={false} mode="popLayout">
            <motion.span
              animate={{ opacity: 1, y: 0 }}
              className="col-start-1 row-start-1 flex items-center gap-1.5"
              exit={{ opacity: 0, y: -4, transition: spring.fast.exit }}
              initial={{ opacity: 0, y: 4 }}
              key={status}
              transition={spring.fast}
            >
              <Icon aria-hidden="true" className="size-3.5" />
              {WORDS[status]}
            </motion.span>
          </AnimatePresence>
          <span
            aria-hidden="true"
            className="invisible col-start-1 row-start-1 pl-5"
          >
            Copied
          </span>
        </span>
      </button>
    </div>
  );
}
