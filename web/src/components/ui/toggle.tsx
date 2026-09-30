"use client";

import * as TogglePrimitive from "@radix-ui/react-toggle";
import type { ComponentProps } from "react";
import { cn, focusRing } from "@/lib/cn";

/** toggleClasses draw a toolbar toggle: quiet at rest, violet on the violet wash while on or open. */
export const toggleClasses = cn(
  "inline-grid size-control shrink-0 cursor-pointer place-items-center rounded-control text-mute transition-colors duration-80 hover:bg-hover hover:text-ink disabled:pointer-events-none disabled:opacity-40 aria-pressed:bg-accent-wash aria-pressed:text-accent aria-pressed:hover:bg-accent-wash aria-pressed:hover:text-accent aria-expanded:bg-accent-wash aria-expanded:text-accent aria-expanded:hover:bg-accent-wash aria-expanded:hover:text-accent [&_svg]:size-4 [&_svg]:shrink-0",
  focusRing,
);

/** Toggle is shadcn's toggle: a button that stays pressed, for bold, italic and the like. */
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
