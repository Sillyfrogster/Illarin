"use client";

import * as SheetPrimitive from "@radix-ui/react-dialog";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

const Sheet = SheetPrimitive.Root;
const SheetTrigger = SheetPrimitive.Trigger;
const SheetClose = SheetPrimitive.Close;
const SheetTitle = SheetPrimitive.Title;
const SheetDescription = SheetPrimitive.Description;

function SheetContent({
  className,
  children,
  ...props
}: ComponentProps<typeof SheetPrimitive.Content>) {
  return (
    <SheetPrimitive.Portal>
      <SheetPrimitive.Overlay className="fixed inset-0 z-90 bg-ink/35 data-[state=closed]:opacity-0 motion-safe:transition-opacity motion-safe:duration-200" />
      <SheetPrimitive.Content
        className={cn(
          "fixed inset-x-0 top-0 z-90 max-h-dvh overflow-y-auto bg-plane pb-6 font-ui text-ink shadow-popover outline-none inset-ring inset-ring-edge/60",
          "motion-safe:data-[state=closed]:animate-slide-out motion-safe:data-[state=open]:animate-slide-in",
          className,
        )}
        {...props}
      >
        {children}
      </SheetPrimitive.Content>
    </SheetPrimitive.Portal>
  );
}

export {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetTitle,
  SheetTrigger,
};
