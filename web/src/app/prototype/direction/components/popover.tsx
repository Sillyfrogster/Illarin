"use client";

import * as PopoverPrimitive from "@radix-ui/react-popover";
import * as React from "react";

import { cn } from "../ui";

const Popover = PopoverPrimitive.Root;

const PopoverTrigger = PopoverPrimitive.Trigger;

const PopoverAnchor = PopoverPrimitive.Anchor;

const PopoverContent = React.forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Content>
>(({ className, align = "center", sideOffset = 4, ...props }, ref) => (
  <PopoverPrimitive.Portal
    container={
      typeof document === "undefined"
        ? undefined
        : document.querySelector<HTMLElement>("[data-direction]")
    }
  >
    <PopoverPrimitive.Content
      ref={ref}
      align={align}
      sideOffset={sideOffset}
      collisionPadding={16}
      sticky="always"
      className={cn(
        "vd:z-50 vd:max-h-[var(--radix-popover-content-available-height)] vd:w-[min(22rem,calc(100vw-2rem))] vd:overflow-y-auto vd:rounded-plate vd:bg-plane vd:p-5 vd:text-ink vd:shadow-[0_16px_60px_-12px_rgb(0_0_0/0.35)] vd:outline-none",
        className,
      )}
      {...props}
    />
  </PopoverPrimitive.Portal>
));
PopoverContent.displayName = PopoverPrimitive.Content.displayName;

export { Popover, PopoverTrigger, PopoverContent, PopoverAnchor };
