"use client";

import * as TogglePrimitive from "@radix-ui/react-toggle";
import type { ComponentProps } from "react";
import { cn, focusRing } from "@/lib/cn";

/** toggleClasses draw a toolbar toggle: grey on hover, violet on the violet wash while on or open. */
export const toggleClasses = cn(
  "inline-grid size-control shrink-0 cursor-pointer place-items-center rounded-control text-mute transition-colors duration-80 hover:bg-fill hover:text-ink disabled:pointer-events-none disabled:opacity-50 aria-pressed:bg-accent-wash aria-pressed:text-accent aria-expanded:bg-accent-wash aria-expanded:text-accent [&_svg]:size-4 [&_svg]:shrink-0",
  focusRing,
);

/** Toggle is a button that stays pressed, for bold, italic and the like. */
export function Toggle({
  className,
  ...props
}: ComponentProps<typeof TogglePrimitive.Root>) {
  return (
    <TogglePrimitive.Root
      className={cn(toggleClasses, className)}
      data-slot="toggle"
      {...props}
    />
  );
}
