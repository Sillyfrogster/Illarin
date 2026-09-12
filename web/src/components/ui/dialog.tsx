"use client";

import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

const Dialog = DialogPrimitive.Root;
const DialogTrigger = DialogPrimitive.Trigger;
const DialogClose = DialogPrimitive.Close;
const DialogTitle = DialogPrimitive.Title;
const DialogDescription = DialogPrimitive.Description;

/** DialogContent is a centred plate that holds the page still behind it. */
function DialogContent({
  className,
  children,
  ...props
}: ComponentProps<typeof DialogPrimitive.Content>) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="fixed inset-0 z-90 bg-ink/40 backdrop-blur-[2px] data-[state=closed]:opacity-0 motion-safe:transition-opacity motion-safe:duration-200" />
      <DialogPrimitive.Content
        className={cn(
          "fixed inset-4 z-90 m-auto flex h-fit max-h-[calc(100dvh-2rem)] w-auto max-w-[1120px] flex-col overflow-hidden rounded-plate bg-plane font-ui text-ink shadow-popover outline-none inset-ring inset-ring-edge/60 sm:inset-6 sm:max-h-[calc(100dvh-3rem)]",
          "motion-safe:data-[state=closed]:animate-plate-out motion-safe:data-[state=open]:animate-plate-in",
          className,
        )}
        {...props}
      >
        {children}
        <DialogPrimitive.Close
          aria-label="Close"
          className="absolute top-3 right-3 inline-flex size-11 items-center justify-center rounded-full text-mute outline-offset-3 hover:bg-deep hover:text-ink"
        >
          <X aria-hidden="true" className="size-5" />
        </DialogPrimitive.Close>
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

export {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
};
