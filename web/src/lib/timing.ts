/** timing is how framer-motion moves things: quick for controls and small swaps, settle for parts moving into place. */
export const timing = {
  quick: { type: "spring" as const, duration: 0.12, bounce: 0 },
  settle: { type: "spring" as const, duration: 0.24, bounce: 0.1 },
} as const;
