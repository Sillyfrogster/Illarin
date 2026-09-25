"use client";

import { motion, useReducedMotion } from "framer-motion";
import {
  type CSSProperties,
  type ReactNode,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { cn } from "@/lib/cn";

const STAGGER_MS = 70;

/** Anything already on screen has nothing to arrive from. */
const BELOW_THE_FOLD = 0.92;

/** Arrive settles a block into view, and rests visible so nothing is left hidden. */
export function Arrive({
  children,
  className,
  layout = false,
  place = 0,
  quiet = false,
  style,
}: {
  children: ReactNode;
  className?: string;
  layout?: boolean;
  place?: number;
  quiet?: boolean;
  style?: CSSProperties;
}) {
  const node = useRef<HTMLDivElement>(null);
  const [waiting, setWaiting] = useState(false);
  const still = useReducedMotion();

  useLayoutEffect(() => {
    const here = node.current;
    if (!here || still || quiet) return;
    if (
      here.getBoundingClientRect().top <
      window.innerHeight * BELOW_THE_FOLD
    ) {
      setWaiting(false);
      return;
    }

    setWaiting(true);
    const follow = new IntersectionObserver(
      ([seen]) => {
        if (!seen.isIntersecting) return;
        setWaiting(false);
        follow.disconnect();
      },
      { rootMargin: "0px 0px -60px 0px" },
    );
    follow.observe(here);
    return () => follow.disconnect();
  }, [still, quiet]);

  return (
    <motion.div
      className={cn(
        "motion-safe:transition-[opacity,translate] motion-safe:duration-700 motion-safe:ease-[var(--ease-wipe)]",
        waiting && !quiet && "translate-y-[18px] opacity-0",
        className,
      )}
      layout={layout && !still ? "position" : false}
      ref={node}
      style={{ transitionDelay: `${place * STAGGER_MS}ms`, ...style }}
      transition={{ type: "spring", stiffness: 380, damping: 36 }}
    >
      {children}
    </motion.div>
  );
}
