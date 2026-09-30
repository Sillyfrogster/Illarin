"use client";

import { MotionConfig } from "framer-motion";
import type { ReactNode } from "react";
import { TooltipProvider } from "@/components/ui/tooltip";
import { type Origins, OriginsProvider } from "@/lib/origins";

export function Providers({
  origins,
  children,
}: {
  origins: Origins;
  children: ReactNode;
}) {
  return (
    <OriginsProvider site={origins.site}>
      <MotionConfig reducedMotion="user">
        <TooltipProvider>{children}</TooltipProvider>
      </MotionConfig>
    </OriginsProvider>
  );
}
