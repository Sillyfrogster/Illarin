"use client";

import * as DialogPrimitive from "@radix-ui/react-dialog";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { X } from "lucide-react";
import { createContext, type ReactNode, useContext, useId } from "react";
import { cn } from "@/lib/cn";
import { PHONE_WIDTH, useMediaQuery } from "@/lib/use-media-query";

const MORPH = { type: "spring", bounce: 0.1, duration: 0.5 } as const;
const TRIGGER_RADIUS = 14;
const PANEL_RADIUS = 20;

type Expanding = { layoutId: string | undefined; open: boolean };

const ExpandingContext = createContext<Expanding | null>(null);

function useExpanding(): Expanding {
  const context = useContext(ExpandingContext);
  if (!context) {
    throw new Error("Expanding panel parts belong inside ExpandingPanel.");
  }
  return context;
}

/** Grows its trigger into a dialog and shrinks it back on closing. */
function ExpandingPanel({
  children,
  onOpenChange,
  open,
}: {
  children: ReactNode;
  onOpenChange: (open: boolean) => void;
  open: boolean;
}) {
  const still = useReducedMotion();
  const id = useId();

  return (
    <ExpandingContext.Provider
      value={{ layoutId: still ? undefined : `expanding-${id}`, open }}
    >
      <DialogPrimitive.Root onOpenChange={onOpenChange} open={open}>
        {children}
      </DialogPrimitive.Root>
    </ExpandingContext.Provider>
  );
}

/** The card the panel grows out of, which keeps its place while the panel is open. */
function ExpandingPanelTrigger({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  const { layoutId, open } = useExpanding();

  return (
    <DialogPrimitive.Trigger asChild>
      <button
        className={cn(
          "group relative isolate block w-full text-left outline-offset-3",
          className,
        )}
        type="button"
      >
        {open ? null : (
          <motion.span
            aria-hidden="true"
            className="absolute inset-0 -z-1 bg-inset transition-colors duration-200 group-hover:bg-deep motion-reduce:transition-none"
            layoutId={layoutId}
            style={{ borderRadius: TRIGGER_RADIUS }}
            transition={MORPH}
          />
        )}
        <span
          className={cn(
            "block transition-opacity duration-150 motion-reduce:transition-none",
            open && "opacity-0",
          )}
        >
          {children}
        </span>
      </button>
    </DialogPrimitive.Trigger>
  );
}

/** The open surface. Its first heading should be an ExpandingPanelTitle. */
function ExpandingPanelContent({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  const { layoutId, open } = useExpanding();
  const still = useReducedMotion();
  const phone = useMediaQuery(PHONE_WIDTH);

  return (
    <AnimatePresence>
      {open ? (
        <DialogPrimitive.Portal forceMount>
          <DialogPrimitive.Overlay asChild forceMount>
            <motion.div
              animate={{ opacity: 1 }}
              className="fixed inset-0 z-90 bg-ink/35"
              exit={{ opacity: 0 }}
              initial={{ opacity: 0 }}
              transition={{ duration: still ? 0 : 0.2 }}
            />
          </DialogPrimitive.Overlay>
          <DialogPrimitive.Content
            aria-describedby={undefined}
            asChild
            forceMount
          >
            <motion.div
              className={cn(
                "fixed inset-0 z-90 flex max-h-dvh flex-col overflow-hidden bg-plane font-ui text-ink shadow-popover outline-none sm:inset-x-6 sm:top-1/2 sm:bottom-auto sm:mx-auto sm:max-h-[min(88dvh,54rem)] sm:w-[min(100%-3rem,64rem)] sm:-translate-y-1/2",
                className,
              )}
              layoutId={layoutId}
              style={{ borderRadius: phone ? 0 : PANEL_RADIUS }}
              transition={MORPH}
            >
              <motion.div
                animate={{ opacity: 1 }}
                className="flex min-h-0 flex-1 flex-col"
                exit={{
                  opacity: 0,
                  transition: { duration: still ? 0 : 0.08 },
                }}
                initial={{ opacity: 0 }}
                transition={
                  still ? { duration: 0 } : { delay: 0.14, duration: 0.2 }
                }
              >
                {children}
              </motion.div>
            </motion.div>
          </DialogPrimitive.Content>
        </DialogPrimitive.Portal>
      ) : null}
    </AnimatePresence>
  );
}

const ExpandingPanelTitle = DialogPrimitive.Title;

function ExpandingPanelClose({ label }: { label: string }) {
  return (
    <DialogPrimitive.Close
      aria-label={label}
      className="inline-flex size-11 shrink-0 items-center justify-center rounded-full text-mute outline-offset-3 hover:bg-deep hover:text-ink"
    >
      <X aria-hidden="true" className="size-5" />
    </DialogPrimitive.Close>
  );
}

export {
  ExpandingPanel,
  ExpandingPanelClose,
  ExpandingPanelContent,
  ExpandingPanelTitle,
  ExpandingPanelTrigger,
};
