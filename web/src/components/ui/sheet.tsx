"use client";

import * as DialogPrimitive from "@radix-ui/react-dialog";
import { motion } from "framer-motion";
import type { ComponentProps } from "react";
import {
  DIALOG_OFFSET,
  Dialog,
  DialogClose,
  DialogDescription,
  DialogTitle,
  fade,
  OVERLAY,
  useDialogOpen,
} from "@/components/ui/dialog";
import { cn } from "@/lib/cn";
import { spring } from "@/lib/springs";
import { surfaceClasses } from "@/lib/surface-classes";
import { SurfaceProvider, useSurface } from "@/lib/surface-context";
import { usePresence } from "@/lib/use-presence";

/** SheetContent is shadcn's sheet from the bottom edge, a dialog's form on a phone, sliding on the moderate spring. */
function SheetContent({
  className,
  children,
  ...props
}: ComponentProps<typeof DialogPrimitive.Content>) {
  const open = useDialogOpen();
  const level = Math.min(useSurface() + DIALOG_OFFSET, 8);
  const { mounted, onExitComplete } = usePresence(open, spring.moderate);

  if (!mounted) return null;

  return (
    <DialogPrimitive.Portal forceMount>
      <DialogPrimitive.Overlay asChild forceMount>
        <motion.div className={OVERLAY} {...fade(open)} />
      </DialogPrimitive.Overlay>
      <DialogPrimitive.Content asChild forceMount {...props}>
        <motion.div
          animate={{ y: open ? 0 : "100%" }}
          className={cn(
            "fixed inset-x-0 bottom-0 z-90 flex max-h-[92dvh] flex-col overflow-hidden rounded-t-plate font-ui text-ink outline-none",
            surfaceClasses(level),
            className,
          )}
          initial={{ y: "100%" }}
          onAnimationComplete={onExitComplete}
          transition={open ? spring.moderate : spring.moderate.exit}
        >
          <SurfaceProvider value={level}>{children}</SurfaceProvider>
        </motion.div>
      </DialogPrimitive.Content>
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
