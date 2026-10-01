"use client";

import * as DialogPrimitive from "@radix-ui/react-dialog";
import type { ComponentProps } from "react";
import {
  Dialog,
  DialogClose,
  DialogDescription,
  DialogTitle,
  SCRIM,
} from "@/components/ui/dialog";
import { cn } from "@/lib/cn";

/** SheetContent is a dialog for a phone that slides up from the bottom edge. */
function SheetContent({
  className,
  ...props
}: ComponentProps<typeof DialogPrimitive.Content>) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className={SCRIM} />
      <DialogPrimitive.Content
        className={cn(
          "fixed inset-x-0 bottom-0 z-90 flex max-h-[92dvh] flex-col overflow-hidden rounded-t-card bg-plane font-ui text-ink outline-none [--slide-from:0_100%] animate-slide-in data-[state=closed]:animate-slide-out",
          className,
        )}
        {...props}
      />
    </DialogPrimitive.Portal>
  );
}

export {
  Dialog as Sheet,
  DialogClose as SheetClose,
  SheetContent,
  DialogDescription as SheetDescription,
  DialogTitle as SheetTitle,
};
