"use client";

import * as SheetPrimitive from "@radix-ui/react-dialog";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

const Sheet = SheetPrimitive.Root;
const SheetTrigger = SheetPrimitive.Trigger;
const SheetClose = SheetPrimitive.Close;
const SheetTitle = SheetPrimitive.Title;
const SheetDescription = SheetPrimitive.Description;

const SIDES = {
  top: "inset-x-0 top-0 max-h-dvh pb-6 motion-safe:data-[state=closed]:animate-slide-out motion-safe:data-[state=open]:animate-slide-in",
  left: "inset-y-0 left-0 h-dvh w-[18rem] max-w-[85vw] motion-safe:data-[state=closed]:animate-slide-left-out motion-safe:data-[state=open]:animate-slide-left-in",
};

function SheetContent({
  className,
  children,
  side = "top",
  ...props
}: ComponentProps<typeof SheetPrimitive.Content> & {
  side?: keyof typeof SIDES;
}) {
  return (
    <SheetPrimitive.Portal>
      <SheetPrimitive.Overlay className="fixed inset-0 z-90 bg-ink/35 data-[state=closed]:opacity-0 motion-safe:transition-opacity motion-safe:duration-200" />
      <SheetPrimitive.Content
        className={cn(
          "fixed z-90 overflow-y-auto bg-plane font-ui text-ink shadow-popover outline-none inset-ring inset-ring-edge/60",
          SIDES[side],
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
