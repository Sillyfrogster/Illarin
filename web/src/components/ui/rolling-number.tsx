"use client";

import { motion, useReducedMotion } from "framer-motion";

const ROLL = { type: "spring", stiffness: 260, damping: 28 } as const;

/** RollingNumber turns each digit like a counter wheel when the number changes. */
export function RollingNumber({ value }: { value: number }) {
  const still = useReducedMotion();
  const digits = String(Math.max(0, Math.round(value))).split("");
  return (
    <span className="relative inline-flex tabular-nums">
      <span className="sr-only">{value}</span>
      {digits.map((digit, index) => (
        <span
          aria-hidden="true"
          className="relative inline-block h-[1lh] overflow-hidden text-center"
          // biome-ignore lint/suspicious/noArrayIndexKey: A digit's place is its identity, so the wheel in that place turns.
          key={digits.length - index}
        >
          <motion.span
            animate={{ y: `${-Number(digit)}lh` }}
            className="flex flex-col"
            initial={false}
            transition={still ? { duration: 0 } : ROLL}
          >
            {WHEEL.map((one) => (
              <span className="h-[1lh]" key={one}>
                {one}
              </span>
            ))}
          </motion.span>
        </span>
      ))}
    </span>
  );
}

const WHEEL = ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9"];
