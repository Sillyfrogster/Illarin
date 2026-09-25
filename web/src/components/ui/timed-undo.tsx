"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { LoaderCircle, Undo2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";

const SPRING = { type: "spring", stiffness: 250, damping: 22 } as const;

/** TimedUndo offers an undo that counts down its last seconds, then calls onExpire. */
export function TimedUndo({
  busy = false,
  onExpire,
  onUndo,
  seconds,
}: {
  busy?: boolean;
  onExpire: () => void;
  onUndo: () => void;
  seconds: number;
}) {
  const [left, setLeft] = useState(seconds);
  const still = useReducedMotion();
  const expire = useRef(onExpire);
  useEffect(() => {
    expire.current = onExpire;
  }, [onExpire]);

  useEffect(() => {
    if (busy) return;
    if (left <= 0) {
      expire.current();
      return;
    }
    const tick = window.setTimeout(() => setLeft(left - 1), 1000);
    return () => window.clearTimeout(tick);
  }, [busy, left]);

  return (
    <motion.button
      aria-label={`Undo, ${left} ${left === 1 ? "second" : "seconds"} left`}
      className="inline-flex min-h-11 shrink-0 items-center gap-2 overflow-hidden rounded-full bg-accent-wash py-1.5 pr-1.5 pl-3.5 font-ui text-meta font-medium text-accent outline-offset-3 hover:bg-accent-wash/70 disabled:opacity-60"
      disabled={busy}
      layout={!still}
      onClick={onUndo}
      transition={SPRING}
      type="button"
    >
      {busy ? (
        <LoaderCircle
          aria-hidden="true"
          className="size-4 animate-spin motion-reduce:animate-none"
        />
      ) : (
        <Undo2 aria-hidden="true" className="size-4" />
      )}
      Undo
      <span
        aria-hidden="true"
        className="relative inline-flex size-8 items-center justify-center overflow-hidden rounded-full bg-action font-ui text-meta text-on-accent tabular-nums"
      >
        <AnimatePresence initial={false} mode="popLayout">
          <motion.span
            animate={{ opacity: 1, y: 0, scale: 1, filter: "blur(0px)" }}
            exit={
              still
                ? { opacity: 0 }
                : { opacity: 0, y: 14, scale: 0.5, filter: "blur(2px)" }
            }
            initial={
              still
                ? { opacity: 0 }
                : { opacity: 0, y: -14, scale: 0.5, filter: "blur(2px)" }
            }
            key={left}
            transition={
              still
                ? { duration: 0 }
                : { ...SPRING, stiffness: 240, damping: 20 }
            }
          >
            {Math.max(left, 0)}
          </motion.span>
        </AnimatePresence>
      </span>
    </motion.button>
  );
}
