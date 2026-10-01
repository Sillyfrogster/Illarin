"use client";

import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ComponentProps } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";

/** SCRIM is the dimmed page behind a dialog, alert dialog, sheet or drawer. */
const SCRIM =
  "fixed inset-0 z-90 bg-scrim animate-fade data-[state=closed]:animate-leave";

/** PANEL is a dialog or alert dialog's plane, centred over the scrim. */
const PANEL =
  "fixed left-1/2 z-90 flex max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] max-w-[540px] -translate-x-1/2 flex-col overflow-hidden rounded-card bg-plane font-ui text-ink ring-1 ring-ink/8 outline-none animate-pop data-[state=closed]:animate-leave focus-visible:outline-none";

const Dialog = DialogPrimitive.Root;
const DialogTrigger = DialogPrimitive.Trigger;
const DialogClose = DialogPrimitive.Close;
const DialogTitle = DialogPrimitive.Title;
const DialogDescription = DialogPrimitive.Description;

/** DialogContent is a flat plane over the dimmed page that closes with Escape or its X. */
function DialogContent({
  className,
  children,
  position = "center",
  showCloseButton = true,
  ...props
}: ComponentProps<typeof DialogPrimitive.Content> & {
  position?: "center" | "top";
  showCloseButton?: boolean;
}) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className={SCRIM} />
      <DialogPrimitive.Content
        className={cn(
          PANEL,
          position === "top" ? "top-[12dvh]" : "top-1/2 -translate-y-1/2",
          className,
        )}
        onOpenAutoFocus={(event) => {
          event.preventDefault();
          const panel = event.currentTarget as HTMLElement;
          if (!panel.contains(document.activeElement)) panel.focus();
        }}
        {...props}
      >
        {children}
        {showCloseButton ? (
          <DialogPrimitive.Close asChild>
            <Button
              aria-label="Close"
              className="absolute top-3 right-3"
              size="icon-compact"
              variant="ghost"
            >
              <X aria-hidden="true" />
            </Button>
          </DialogPrimitive.Close>
        ) : null}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

export {
  PANEL,
  SCRIM,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
};
