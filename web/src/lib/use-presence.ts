"use client";

import { useEffect, useState } from "react";
import { exitFallbackMs } from "@/lib/springs";

/** usePresence keeps a closing popup mounted until its exit animation ends, with a timer in case a background tab stalls it. */
export function usePresence(
  open: boolean,
  tier: { exit: { duration: number } },
) {
  const [mounted, setMounted] = useState(open);
  useEffect(() => {
    if (open) {
      setMounted(true);
      return;
    }
    const id = setTimeout(() => setMounted(false), exitFallbackMs(tier));
    return () => clearTimeout(id);
  }, [open, tier]);
  return {
    mounted,
    onExitComplete: () => {
      if (!open) setMounted(false);
    },
  };
}
