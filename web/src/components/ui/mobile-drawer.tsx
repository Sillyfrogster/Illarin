"use client";

import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ReactNode, RefObject } from "react";
import { Button } from "@/components/ui/button";
import { SCRIM } from "@/components/ui/dialog";

/** MobileDrawer is the phone navigation panel, sliding in from the left and handing focus back to its trigger. */
export function MobileDrawer({
  open,
  onClose,
  title,
  children,
  triggerRef,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  triggerRef?: RefObject<HTMLElement | null>;
}) {
  return (
    <DialogPrimitive.Root
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      open={open}
    >
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className={SCRIM} />
        <DialogPrimitive.Content
          aria-describedby={undefined}
          className="fixed inset-y-0 left-0 z-90 flex w-72 max-w-[85vw] flex-col overflow-y-auto bg-plane font-ui text-ink outline-none [--slide-from:-100%_0] animate-slide-in data-[state=closed]:animate-slide-out"
          onCloseAutoFocus={(event) => {
            if (!triggerRef?.current) return;
            event.preventDefault();
            triggerRef.current.focus();
          }}
        >
          <DialogPrimitive.Title className="sr-only">
            {title}
          </DialogPrimitive.Title>
          {children}
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
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
