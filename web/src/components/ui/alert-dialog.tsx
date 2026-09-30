"use client";

import * as AlertDialogPrimitive from "@radix-ui/react-alert-dialog";
import { motion } from "framer-motion";
import {
  type ComponentProps,
  createContext,
  type ReactNode,
  useContext,
} from "react";
import {
  DIALOG_OFFSET,
  fade,
  OVERLAY,
  PANEL,
  rise,
} from "@/components/ui/dialog";
import { cn } from "@/lib/cn";
import { spring } from "@/lib/springs";
import { surfaceClasses } from "@/lib/surface-classes";
import { SurfaceProvider, useSurface } from "@/lib/surface-context";
import { usePresence } from "@/lib/use-presence";

const AlertOpenContext = createContext(false);

/** AlertDialog confirms what cannot be undone; focus starts on the safe choice and Escape cancels. */
function AlertDialog({
  open,
  onOpenChange,
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
}) {
  return (
    <AlertOpenContext.Provider value={open}>
      <AlertDialogPrimitive.Root onOpenChange={onOpenChange} open={open}>
        {children}
      </AlertDialogPrimitive.Root>
    </AlertOpenContext.Provider>
  );
}

/** AlertDialogContent is shadcn's alert dialog on the dialog's surface and slow spring. */
function AlertDialogContent({
  className,
  children,
  ...props
}: ComponentProps<typeof AlertDialogPrimitive.Content>) {
  const open = useContext(AlertOpenContext);
  const level = Math.min(useSurface() + DIALOG_OFFSET, 8);
  const { mounted, onExitComplete } = usePresence(open, spring.slow);
  if (!mounted) return null;
  return (
    <AlertDialogPrimitive.Portal forceMount>
      <AlertDialogPrimitive.Overlay asChild forceMount>
        <motion.div className={OVERLAY} {...fade(open)} />
      </AlertDialogPrimitive.Overlay>
      <AlertDialogPrimitive.Content asChild forceMount {...props}>
        <motion.div
          className={cn(
            PANEL,
            "top-1/2 max-w-[32rem] gap-3 p-6",
            surfaceClasses(level),
            className,
          )}
          onAnimationComplete={onExitComplete}
          {...rise(open, "-50%")}
        >
          <SurfaceProvider value={level}>{children}</SurfaceProvider>
        </motion.div>
      </AlertDialogPrimitive.Content>
    </AlertDialogPrimitive.Portal>
  );
}

function AlertDialogTitle({
  className,
  ...props
}: ComponentProps<typeof AlertDialogPrimitive.Title>) {
  return (
    <AlertDialogPrimitive.Title
      className={cn(
        "font-display text-section font-medium text-ink",
        className,
      )}
      {...props}
    />
  );
}

function AlertDialogDescription({
  className,
  ...props
}: ComponentProps<typeof AlertDialogPrimitive.Description>) {
  return (
    <AlertDialogPrimitive.Description
      className={cn("text-ui text-mute", className)}
      {...props}
    />
  );
}

function AlertDialogFooter({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("mt-3 flex flex-wrap justify-end gap-2", className)}
      {...props}
    />
  );
}

const AlertDialogCancel = AlertDialogPrimitive.Cancel;
const AlertDialogAction = AlertDialogPrimitive.Action;

export {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
};
