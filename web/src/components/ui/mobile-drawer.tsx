"use client";

import * as DialogPrimitive from "@radix-ui/react-dialog";
import { motion } from "framer-motion";
import type { ReactNode, RefObject } from "react";
import { spring } from "@/lib/springs";
import { surfaceClasses } from "@/lib/surface-classes";
import { SurfaceProvider, useSurface } from "@/lib/surface-context";
import { usePresence } from "@/lib/use-presence";

/** MobileDrawer is Fluid Functionalism's navigation panel for a phone: it slides in from the left on the moderate spring and hands focus back to its trigger. */
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
  const level = Math.min(useSurface() + 2, 8);
  const { mounted, onExitComplete } = usePresence(open, spring.moderate);

  return (
    <DialogPrimitive.Root
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      open={open}
    >
      {mounted ? (
        <DialogPrimitive.Portal forceMount>
          <DialogPrimitive.Overlay asChild forceMount>
            <motion.div
              animate={{ opacity: open ? 1 : 0 }}
              className="fixed inset-0 z-90 bg-[rgb(0_0_0/0.4)] dark:bg-[rgb(0_0_0/0.7)]"
              initial={{ opacity: 0 }}
              transition={open ? spring.moderate : spring.moderate.exit}
            />
          </DialogPrimitive.Overlay>
          <DialogPrimitive.Content
            aria-describedby={undefined}
            asChild
            forceMount
            onCloseAutoFocus={(event) => {
              if (!triggerRef?.current) return;
              event.preventDefault();
              triggerRef.current.focus();
            }}
          >
            <motion.div
              animate={{ x: open ? 0 : "-100%" }}
              className={`fixed inset-y-0 left-0 z-90 flex w-72 max-w-[85vw] flex-col overflow-y-auto font-ui text-ink outline-none ${surfaceClasses(level, 3)}`}
              initial={{ x: "-100%" }}
              onAnimationComplete={onExitComplete}
              transition={open ? spring.moderate : spring.moderate.exit}
            >
              <DialogPrimitive.Title className="sr-only">
                {title}
              </DialogPrimitive.Title>
              <SurfaceProvider value={level}>{children}</SurfaceProvider>
            </motion.div>
          </DialogPrimitive.Content>
        </DialogPrimitive.Portal>
      ) : null}
    </DialogPrimitive.Root>
  );
}
