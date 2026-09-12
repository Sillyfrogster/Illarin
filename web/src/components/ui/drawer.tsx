"use client";

import type { ComponentProps } from "react";
import { Drawer as DrawerPrimitive } from "vaul";
import { cn } from "@/lib/cn";

const Drawer = DrawerPrimitive.Root;
const DrawerClose = DrawerPrimitive.Close;
const DrawerTitle = DrawerPrimitive.Title;

/** DrawerContent rises from the bottom edge and follows a drag back down. */
function DrawerContent({
  className,
  children,
  ...props
}: ComponentProps<typeof DrawerPrimitive.Content>) {
  return (
    <DrawerPrimitive.Portal>
      <DrawerPrimitive.Overlay className="fixed inset-0 z-90 bg-ink/35" />
      <DrawerPrimitive.Content
        className={cn(
          "fixed inset-x-0 bottom-0 z-90 flex max-h-[88dvh] flex-col rounded-t-plate bg-plane font-ui text-ink shadow-popover outline-none inset-ring inset-ring-edge/60",
          className,
        )}
        {...props}
      >
        <div
          aria-hidden="true"
          className="mx-auto mt-3 h-1.5 w-12 shrink-0 rounded-full bg-rule"
        />
        {children}
      </DrawerPrimitive.Content>
    </DrawerPrimitive.Portal>
  );
}

export { Drawer, DrawerClose, DrawerContent, DrawerTitle };
