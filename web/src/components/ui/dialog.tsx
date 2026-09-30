"use client";

import * as DialogPrimitive from "@radix-ui/react-dialog";
import { motion } from "framer-motion";
import { X } from "lucide-react";
import {
  type ComponentProps,
  createContext,
  useContext,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";
import { spring } from "@/lib/springs";
import { surfaceClasses } from "@/lib/surface-classes";
import { SurfaceProvider, useSurface } from "@/lib/surface-context";
import { usePresence } from "@/lib/use-presence";

const DIALOG_OFFSET = 4;

/** OVERLAY is the dimmed page behind a dialog or alert dialog. */
const OVERLAY =
  "fixed inset-0 z-90 bg-[rgb(0_0_0/0.4)] dark:bg-[rgb(0_0_0/0.7)]";

/** PANEL is the raised surface of a dialog or alert dialog, centred on the page. */
const PANEL =
  "fixed left-1/2 z-90 flex max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] max-w-[540px] flex-col overflow-hidden rounded-plate font-ui text-ink outline-none focus-visible:outline-none";

/** fade and rise are the dialog's enter and exit on the slow spring. */
function fade(open: boolean) {
  return {
    initial: { opacity: 0 },
    animate: { opacity: open ? 1 : 0 },
    transition: open ? spring.slow : spring.slow.exit,
  };
}

function rise(open: boolean, y: number | string) {
  return {
    initial: { opacity: 0, scale: 0.97, x: "-50%", y },
    animate: { opacity: open ? 1 : 0, scale: open ? 1 : 0.97, x: "-50%", y },
    transition: open ? spring.slow : spring.slow.exit,
  };
}

const DialogOpenContext = createContext(false);

/** Dialog keeps its open state so the panel can play its exit before it unmounts. */
function Dialog({
  children,
  open: controlledOpen,
  defaultOpen,
  onOpenChange,
  ...props
}: DialogPrimitive.DialogProps) {
  const [uncontrolledOpen, setUncontrolledOpen] = useState(
    defaultOpen ?? false,
  );
  const open = controlledOpen ?? uncontrolledOpen;
  return (
    <DialogOpenContext.Provider value={open}>
      <DialogPrimitive.Root
        onOpenChange={(next) => {
          setUncontrolledOpen(next);
          onOpenChange?.(next);
        }}
        open={open}
        {...props}
      >
        {children}
      </DialogPrimitive.Root>
    </DialogOpenContext.Provider>
  );
}

const DialogTrigger = DialogPrimitive.Trigger;
const DialogClose = DialogPrimitive.Close;
const DialogTitle = DialogPrimitive.Title;
const DialogDescription = DialogPrimitive.Description;

/** DialogContent is Fluid Functionalism's dialog: a raised panel that scales in on the slow spring over a dimmed page. */
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
  const open = useContext(DialogOpenContext);
  const level = Math.min(useSurface() + DIALOG_OFFSET, 8);
  const { mounted, onExitComplete } = usePresence(open, spring.slow);
  const y = position === "top" ? 0 : "-50%";

  if (!mounted) return null;

  return (
    <DialogPrimitive.Portal forceMount>
      <DialogPrimitive.Overlay asChild forceMount>
        <motion.div className={OVERLAY} {...fade(open)} />
      </DialogPrimitive.Overlay>
      <DialogPrimitive.Content
        asChild
        forceMount
        onOpenAutoFocus={(event) => {
          event.preventDefault();
          const panel = event.currentTarget as HTMLElement;
          if (!panel.contains(document.activeElement)) panel.focus();
        }}
        {...props}
      >
        <motion.div
          className={cn(
            PANEL,
            position === "top" ? "top-[12dvh]" : "top-1/2",
            surfaceClasses(level),
            className,
          )}
          onAnimationComplete={onExitComplete}
          {...rise(open, y)}
        >
          <SurfaceProvider value={level}>
            {children}
            {showCloseButton ? (
              <DialogPrimitive.Close asChild>
                <Button
                  className="absolute top-3 right-3"
                  size="icon-compact"
                  variant="ghost"
                >
                  <X aria-hidden="true" />
                  <span className="sr-only">Close</span>
                </Button>
              </DialogPrimitive.Close>
            ) : null}
          </SurfaceProvider>
        </motion.div>
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

export {
  DIALOG_OFFSET,
  fade,
  OVERLAY,
  PANEL,
  rise,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
};
