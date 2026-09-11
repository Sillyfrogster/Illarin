"use client";

import { MotionConfig } from "framer-motion";
import type { ReactNode } from "react";
import { type Origins, OriginsProvider } from "@/lib/origins";

export function Providers({
  origins,
  children,
}: {
  origins: Origins;
  children: ReactNode;
}) {
  return (
    <OriginsProvider blog={origins.blog} site={origins.site}>
      <MotionConfig reducedMotion="user">{children}</MotionConfig>
    </OriginsProvider>
  );
}
