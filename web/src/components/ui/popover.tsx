"use client";

import * as PopoverPrimitive from "@radix-ui/react-popover";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

const Popover = PopoverPrimitive.Root;
const PopoverTrigger = PopoverPrimitive.Trigger;
const PopoverAnchor = PopoverPrimitive.Anchor;

/** An open task, lifted off the page rather than pushed into it */
function PopoverContent({
  className,
  align = "center",
  sideOffset = 10,
  ...props
}: ComponentProps<typeof PopoverPrimitive.Content>) {
  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Content
        align={align}
        sideOffset={sideOffset}
        collisionPadding={16}
        className={cn(
          "z-90 max-h-[var(--radix-popover-content-available-height)] w-[min(22rem,calc(100vw-2rem))] origin-[var(--radix-popover-content-transform-origin)] overflow-y-auto rounded-plate bg-plane p-5 font-ui text-ui text-ink shadow-popover outline-none",
          "inset-ring inset-ring-rule/70 motion-safe:data-[state=closed]:animate-pop-out motion-safe:data-[state=open]:animate-pop-in",
          className,
        )}
        {...props}
      />
    </PopoverPrimitive.Portal>
  );
}

export { Popover, PopoverAnchor, PopoverContent, PopoverTrigger };
