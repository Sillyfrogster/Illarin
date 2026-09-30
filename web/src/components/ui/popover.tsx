"use client";

import * as PopoverPrimitive from "@radix-ui/react-popover";
import { motion } from "framer-motion";
import {
  type ComponentProps,
  createContext,
  type ReactNode,
  useContext,
  useState,
} from "react";
import { cn } from "@/lib/cn";
import { Elevated } from "@/lib/elevated";
import { popupMotionClass } from "@/lib/popup";
import { spring } from "@/lib/springs";
import { usePresence } from "@/lib/use-presence";

const PopoverOpenContext = createContext(false);

/** Popover keeps its open state so the panel can play its exit before it unmounts. */
function Popover({
  open: openProp,
  defaultOpen = false,
  onOpenChange,
  children,
}: {
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  children: ReactNode;
}) {
  const [inner, setInner] = useState(defaultOpen);
  const open = openProp ?? inner;
  return (
    <PopoverOpenContext.Provider value={open}>
      <PopoverPrimitive.Root
        onOpenChange={(next) => {
          if (openProp === undefined) setInner(next);
          onOpenChange?.(next);
        }}
        open={open}
      >
        {children}
      </PopoverPrimitive.Root>
    </PopoverOpenContext.Provider>
  );
}

const PopoverTrigger = PopoverPrimitive.Trigger;
const PopoverAnchor = PopoverPrimitive.Anchor;

/** PopoverContent is a panel anchored to a control, on Fluid Functionalism's popup surface and fast spring. */
function PopoverContent({
  className,
  children,
  align = "center",
  sideOffset = 6,
  collisionPadding = 16,
  ...props
}: ComponentProps<typeof PopoverPrimitive.Content>) {
  const open = useContext(PopoverOpenContext);
  const { mounted, onExitComplete } = usePresence(open, spring.fast);
  if (!mounted) return null;
  return (
    <PopoverPrimitive.Portal forceMount>
      <PopoverPrimitive.Content
        align={align}
        asChild
        collisionPadding={collisionPadding}
        forceMount
        sideOffset={sideOffset}
        {...props}
      >
        <motion.div
          animate={
            open
              ? { opacity: 1, y: 0, scaleY: 1 }
              : { opacity: 0, y: "var(--popup-enter-y)", scaleY: 0.96 }
          }
          className={cn("z-90 outline-none", popupMotionClass)}
          initial={{ opacity: 0, y: "var(--popup-enter-y)", scaleY: 0.96 }}
          onAnimationComplete={onExitComplete}
          transition={open ? spring.fast : spring.fast.exit}
        >
          <Elevated
            className={cn(
              "max-h-(--radix-popover-content-available-height) w-[min(22rem,calc(100vw-2rem))] overflow-y-auto rounded-plate p-4 font-ui text-ui text-ink",
              className,
            )}
            offset={2}
            shadowLevel={3}
          >
            {children}
          </Elevated>
        </motion.div>
      </PopoverPrimitive.Content>
    </PopoverPrimitive.Portal>
  );
}

export { Popover, PopoverAnchor, PopoverContent, PopoverTrigger };
