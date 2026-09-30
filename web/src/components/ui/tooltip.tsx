"use client";

import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { motion } from "framer-motion";
import {
  createContext,
  type ReactElement,
  type ReactNode,
  useContext,
  useEffect,
  useState,
} from "react";
import { fontWeights } from "@/lib/font-weight";
import { exitFallbackMs, spring } from "@/lib/springs";

const DEFAULT_DELAY = 200;

const TooltipGroupContext = createContext(false);

/** TooltipProvider lets the reader move between neighbouring tooltips without waiting the delay again. */
function TooltipProvider({ children }: { children: ReactNode }) {
  return (
    <TooltipGroupContext.Provider value={true}>
      <TooltipPrimitive.Provider
        delayDuration={DEFAULT_DELAY}
        skipDelayDuration={300}
      >
        {children}
      </TooltipPrimitive.Provider>
    </TooltipGroupContext.Provider>
  );
}

type TooltipSide = "top" | "right" | "bottom" | "left";

const SLIDE: Record<TooltipSide, { x?: number; y?: number }> = {
  top: { y: 4 },
  bottom: { y: -4 },
  left: { x: 4 },
  right: { x: -4 },
};

/** Tooltip names an icon button on hover and keyboard focus, sliding in on the fast spring. */
function Tooltip({
  content,
  children,
  side = "top",
  sideOffset = 8,
}: {
  content: ReactNode;
  children: ReactElement;
  side?: TooltipSide;
  sideOffset?: number;
}) {
  const [open, setOpen] = useState(false);
  const [mounted, setMounted] = useState(false);
  const grouped = useContext(TooltipGroupContext);

  useEffect(() => {
    if (open) {
      setMounted(true);
      return;
    }
    const id = setTimeout(() => setMounted(false), exitFallbackMs(spring.fast));
    return () => clearTimeout(id);
  }, [open]);

  const tooltip = (
    <TooltipPrimitive.Root onOpenChange={setOpen} open={open}>
      <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
      {mounted ? (
        <TooltipPrimitive.Portal forceMount>
          <TooltipPrimitive.Content
            className="z-100"
            forceMount
            side={side}
            sideOffset={sideOffset}
          >
            <motion.div
              animate={{ opacity: open ? 1 : 0, x: 0, y: 0 }}
              className="max-w-64 rounded-control bg-ink px-2 py-1 font-ui text-label text-field"
              initial={{ opacity: 0, ...SLIDE[side] }}
              onAnimationComplete={() => {
                if (!open) setMounted(false);
              }}
              style={{ fontVariationSettings: fontWeights.medium }}
              transition={open ? spring.fast : spring.fast.exit}
            >
              {content}
            </motion.div>
          </TooltipPrimitive.Content>
        </TooltipPrimitive.Portal>
      ) : null}
    </TooltipPrimitive.Root>
  );

  if (grouped) return tooltip;
  return (
    <TooltipPrimitive.Provider delayDuration={DEFAULT_DELAY}>
      {tooltip}
    </TooltipPrimitive.Provider>
  );
}

export { Tooltip, TooltipProvider };
