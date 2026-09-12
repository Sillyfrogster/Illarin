"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { ChevronDown } from "lucide-react";
import { type ReactNode, useId, useState } from "react";
import { cn } from "@/lib/cn";

const MORPH = { type: "spring", stiffness: 500, damping: 40 } as const;

export function MorphingDisclosure({
  summary,
  children,
  className,
  lead,
  trailing,
}: {
  summary: ReactNode;
  children: ReactNode;
  className?: string;
  lead?: ReactNode;
  trailing?: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const still = useReducedMotion();
  const panel = useId();

  return (
    <motion.div
      className={cn("overflow-hidden", className)}
      layout={still ? false : "size"}
      transition={MORPH}
    >
      <div className="flex flex-wrap items-center justify-between gap-x-8 gap-y-3">
        {lead}
        <button
          aria-controls={panel}
          aria-expanded={open}
          className="group inline-flex min-h-11 items-center gap-2 text-ui font-medium text-ink outline-offset-3"
          onClick={() => setOpen(!open)}
          type="button"
        >
          <ChevronDown
            aria-hidden="true"
            className={cn(
              "size-4 text-accent transition-transform duration-200 motion-reduce:transition-none",
              open ? "rotate-0" : "-rotate-90",
            )}
          />
          <span className="underline decoration-transparent decoration-1 underline-offset-4 transition-colors group-hover:decoration-accent motion-reduce:transition-none">
            {summary}
          </span>
        </button>
        {trailing}
      </div>

      <AnimatePresence initial={false}>
        {open ? (
          <motion.div
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: still ? 0 : -6 }}
            id={panel}
            initial={{ opacity: 0, y: still ? 0 : 8 }}
            key="panel"
            layout={still ? false : "position"}
            transition={
              still
                ? { duration: 0 }
                : { duration: 0.3, ease: [0.22, 1, 0.36, 1] }
            }
          >
            {children}
          </motion.div>
        ) : null}
      </AnimatePresence>
    </motion.div>
  );
}
