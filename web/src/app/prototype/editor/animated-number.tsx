"use client";

import { motion, useReducedMotion } from "framer-motion";

const digits = Array.from({ length: 10 }, (_, digit) => digit);

/** Adapted from Vengeance UI's digit-strip animation. See THIRD_PARTY.md. */
export function AnimatedNumber({ value }: { value: number }) {
  const reduced = useReducedMotion();
  const places = String(value)
    .split("")
    .map((digit, index, all) => ({ digit, place: all.length - index }));
  return (
    <span className="ws:inline-flex ws:tabular-nums">
      <span className="ws:sr-only">{value}</span>
      <span
        aria-hidden="true"
        className="ws:inline-flex ws:h-[1em] ws:overflow-hidden ws:leading-none"
      >
        {places.map(({ digit, place }) => (
          <span
            key={place}
            className="ws:relative ws:inline-block ws:w-[0.65em]"
          >
            <motion.span
              initial={false}
              animate={{ y: `${-Number(digit)}em` }}
              transition={{ duration: reduced ? 0 : 0.25, ease: "easeOut" }}
              className="ws:absolute ws:top-0 ws:left-0 ws:flex ws:flex-col"
            >
              {digits.map((number) => (
                <span key={number} className="ws:h-[1em]">
                  {number}
                </span>
              ))}
            </motion.span>
          </span>
        ))}
      </span>
    </span>
  );
}
