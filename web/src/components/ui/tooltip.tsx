"use client";

import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import {
  createContext,
  type ReactElement,
  type ReactNode,
  useContext,
} from "react";

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

/** Tooltip names an icon button on hover and keyboard focus, flipping to the other side near a screen edge. */
function Tooltip({
  content,
  children,
  side = "top",
  sideOffset = 8,
}: {
  content: ReactNode;
  children: ReactElement;
  side?: "top" | "right" | "bottom" | "left";
  sideOffset?: number;
}) {
  const tooltip = (
    <TooltipPrimitive.Root>
      <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
      <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
          className="z-100 max-w-64 animate-pop rounded-chip bg-ink px-2 py-1 font-ui text-label font-medium text-field data-[state=closed]:animate-leave"
          collisionPadding={8}
          side={side}
          sideOffset={sideOffset}
        >
          {content}
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  );

  if (useContext(TooltipGroupContext)) return tooltip;
  return (
    <TooltipPrimitive.Provider delayDuration={DEFAULT_DELAY}>
      {tooltip}
    </TooltipPrimitive.Provider>
  );
}

export { Tooltip, TooltipProvider };
