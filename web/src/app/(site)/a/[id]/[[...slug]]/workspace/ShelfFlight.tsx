"use client";

import { motion, useReducedMotion } from "framer-motion";
import { useEffect } from "react";

type Box = { left: number; top: number; width: number; height: number };

export type Flight = { from: Box; to: Box; title: string };

/** ShelfFlight carries a dropped section from where the pointer let go into the place it lands. */
export function ShelfFlight({
  flight,
  onLanded,
}: {
  flight: Flight | null;
  onLanded: () => void;
}) {
  const still = useReducedMotion();

  useEffect(() => {
    if (flight && still) onLanded();
  }, [flight, still, onLanded]);

  if (!flight || still) return null;
  const { from, to } = flight;
  return (
    <motion.div
      animate={{
        x: to.left,
        y: to.top,
        width: to.width,
        height: Math.min(to.height, 160),
        rotate: 0,
        scale: 1,
        opacity: [1, 1, 0],
      }}
      aria-hidden="true"
      className="pointer-events-none fixed top-0 left-0 z-60 overflow-hidden rounded-plate bg-plane px-5 py-4 shadow-popover inset-ring-2 inset-ring-accent"
      initial={{
        x: from.left,
        y: from.top,
        width: from.width,
        height: from.height,
        rotate: -1.5,
        scale: 1.03,
        opacity: 1,
      }}
      key={`${to.left}:${to.top}:${flight.title}`}
      onAnimationComplete={onLanded}
      transition={{
        duration: 0.42,
        ease: [0.22, 1, 0.36, 1],
        opacity: { duration: 0.42, times: [0, 0.75, 1] },
      }}
    >
      <p className="font-display text-ui font-medium text-ink wrap-anywhere">
        {flight.title}
      </p>
    </motion.div>
  );
}
