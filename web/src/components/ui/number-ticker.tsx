"use client";

import { useMotionValue, useReducedMotion, useSpring } from "framer-motion";
import { type ComponentProps, useEffect, useRef } from "react";
import { cn } from "@/lib/cn";
import { spring } from "@/lib/springs";

const format = new Intl.NumberFormat("en-US");

/** NumberTicker is Magic UI's number ticker: it starts at the value it is given and counts to each new one on the slow spring. */
export function NumberTicker({
  value,
  className,
  ...props
}: ComponentProps<"span"> & { value: number }) {
  const shown = useRef<HTMLSpanElement>(null);
  const still = useReducedMotion();
  const motionValue = useMotionValue(value);
  const springValue = useSpring(motionValue, {
    duration: spring.slow.duration,
    bounce: 0,
  });

  useEffect(() => {
    if (still) springValue.jump(value);
    motionValue.set(value);
  }, [motionValue, springValue, still, value]);

  useEffect(
    () =>
      springValue.on("change", (latest) => {
        if (shown.current)
          shown.current.textContent = format.format(Math.round(latest));
      }),
    [springValue],
  );

  return (
    <span className={cn("inline-block tabular-nums", className)} {...props}>
      <span className="sr-only">{format.format(value)}</span>
      <span aria-hidden="true" ref={shown}>
        {format.format(value)}
      </span>
    </span>
  );
}
