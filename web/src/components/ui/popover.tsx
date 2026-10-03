"use client";

import * as PopoverPrimitive from "@radix-ui/react-popover";
import type { ComponentProps } from "react";
import { cn, popupSurface } from "@/lib/cn";

const Popover = PopoverPrimitive.Root;
const PopoverTrigger = PopoverPrimitive.Trigger;
const PopoverAnchor = PopoverPrimitive.Anchor;

/** PopoverContent is a panel anchored to a control, on the same plane as a menu. */
function PopoverContent({
  className,
  align = "center",
  sideOffset = 6,
  collisionPadding = 16,
  ...props
}: ComponentProps<typeof PopoverPrimitive.Content>) {
  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Content
        align={align}
        className={cn(
          popupSurface,
          "max-h-(--radix-popover-content-available-height) w-[min(22rem,calc(100vw-2rem))] overflow-y-auto p-4 text-ui",
          className,
        )}
        collisionPadding={collisionPadding}
        sideOffset={sideOffset}
        {...props}
      />
    </PopoverPrimitive.Portal>
  );
}

export { Popover, PopoverAnchor, PopoverContent, PopoverTrigger };
