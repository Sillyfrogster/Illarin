"use client";

import { useReducedMotion } from "framer-motion";
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
  place = 0,
  style,
}: {
  children: ReactNode;
  className?: string;
  place?: number;
  style?: CSSProperties;
}) {
  const node = useRef<HTMLDivElement>(null);
  const [waiting, setWaiting] = useState(false);
  const still = useReducedMotion();

  useLayoutEffect(() => {
    const here = node.current;
    if (!here || still) return;
    if (
      here.getBoundingClientRect().top <
      window.innerHeight * BELOW_THE_FOLD
    ) {
      return;
    }

    setWaiting(true);
    const watch = new IntersectionObserver(
      ([seen]) => {
        if (!seen.isIntersecting) return;
        setWaiting(false);
        watch.disconnect();
      },
      { rootMargin: "0px 0px -60px 0px" },
    );
    watch.observe(here);
    return () => watch.disconnect();
  }, [still]);

  return (
    <div
      className={cn(
        "motion-safe:transition-[opacity,translate] motion-safe:duration-700 motion-safe:ease-[var(--ease-wipe)]",
        waiting && "translate-y-[18px] opacity-0",
        className,
      )}
      ref={node}
      style={{ transitionDelay: `${place * STAGGER_MS}ms`, ...style }}
    >
      {children}
    </div>
  );
}
