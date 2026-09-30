"use client";

import * as DropdownMenuPrimitive from "@radix-ui/react-dropdown-menu";
import { motion } from "framer-motion";
import { Check } from "lucide-react";
import {
  type ComponentProps,
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/cn";
import { Elevated } from "@/lib/elevated";
import {
  isDisabledRow,
  popupMotionClass,
  popupScrollAreaClass,
  popupViewportClass,
} from "@/lib/popup";
import { exitFallbackMs, spring } from "@/lib/springs";
import {
  useFluidHover,
  useRegisterFluidHoverItem,
} from "@/lib/use-fluid-hover";

const MenuOpenContext = createContext(false);

type Rows = {
  register: (index: number, element: HTMLElement | null) => void;
  next: () => number;
  light: (index: number) => void;
};

const RowsContext = createContext<Rows | null>(null);

/** DropdownMenu keeps its open state so the popup can play its exit before it unmounts; it never locks the page. */
function DropdownMenu({
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
  const change = useCallback(
    (next: boolean) => {
      if (openProp === undefined) setInner(next);
      onOpenChange?.(next);
    },
    [openProp, onOpenChange],
  );
  return (
    <MenuOpenContext.Provider value={open}>
      <DropdownMenuPrimitive.Root
        modal={false}
        onOpenChange={change}
        open={open}
      >
        {children}
      </DropdownMenuPrimitive.Root>
    </MenuOpenContext.Provider>
  );
}

const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;
const DropdownMenuGroup = DropdownMenuPrimitive.Group;
const DropdownMenuRadioGroup = DropdownMenuPrimitive.RadioGroup;

/** DropdownMenuContent is Fluid Functionalism's dropdown popup: a raised surface that grows from its trigger, with one plate that follows the pointer between rows. */
function DropdownMenuContent({
  className,
  children,
  align = "start",
  sideOffset = 6,
  collisionPadding = 16,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Content>) {
  const open = useContext(MenuOpenContext);
  const [mounted, setMounted] = useState(open);
  const containerRef = useRef<HTMLDivElement>(null);
  const hover = useFluidHover(containerRef, { isItemDisabled: isDisabledRow });
  const { registerItem, setActiveIndex, remeasure, handlers } = hover;
  const counter = useRef(0);

  useEffect(() => {
    if (open) {
      setMounted(true);
      return;
    }
    const id = setTimeout(() => setMounted(false), exitFallbackMs(spring.fast));
    return () => clearTimeout(id);
  }, [open]);

  useEffect(() => {
    if (open && mounted) remeasure();
  }, [open, mounted, remeasure]);

  const rows = useMemo<Rows>(
    () => ({
      register: registerItem,
      next: () => counter.current++,
      light: setActiveIndex,
    }),
    [registerItem, setActiveIndex],
  );

  if (!mounted) return null;

  return (
    <DropdownMenuPrimitive.Portal forceMount>
      <DropdownMenuPrimitive.Content
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
          onAnimationComplete={() => {
            if (!open) setMounted(false);
          }}
          transition={open ? spring.fast : spring.fast.exit}
        >
          <RowsContext.Provider value={rows}>
            <Elevated
              className={cn(
                "flex max-h-[min(480px,var(--radix-dropdown-menu-content-available-height))] w-max max-w-[calc(100vw-2rem)] min-w-[max(12rem,var(--radix-dropdown-menu-trigger-width))] flex-col overflow-hidden rounded-plate font-ui text-ink select-none",
                className,
              )}
              offset={2}
              onClick={handlers.onClick}
              onMouseEnter={handlers.onMouseEnter}
              onMouseLeave={handlers.onMouseLeave}
              onMouseMove={handlers.onMouseMove}
              shadowLevel={3}
            >
              <ScrollArea
                className={popupScrollAreaClass}
                viewportClassName={cn(popupViewportClass, "scroll-fade")}
              >
                <div className="relative flex flex-col p-1" ref={containerRef}>
                  <FluidHoverHighlight
                    className="rounded-control"
                    hover={hover}
                  />
                  {children}
                </div>
              </ScrollArea>
            </Elevated>
          </RowsContext.Provider>
        </motion.div>
      </DropdownMenuPrimitive.Content>
    </DropdownMenuPrimitive.Portal>
  );
}

/** useRow registers a menu row with the popup's hover plate, which also follows keyboard focus. */
function useRow() {
  const rows = useContext(RowsContext);
  const [index] = useState(() => rows?.next() ?? 0);
  const ref = useRef<HTMLElement>(null);
  useRegisterFluidHoverItem(rows?.register, rows ? index : undefined, ref);
  return {
    ref,
    "data-fluid-hover-index": index,
    onFocus: () => rows?.light(index),
  };
}

const ROW =
  "group/row relative z-10 flex min-h-control shrink-0 cursor-pointer items-center gap-2 rounded-control px-2 font-ui text-ui text-mute outline-none transition-colors focus-visible:outline-none duration-80 select-none data-[highlighted]:text-ink data-[current=page]:text-accent data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0 [&_svg]:stroke-[1.5] data-[highlighted]:[&_svg]:stroke-2";

function DropdownMenuItem({
  className,
  onFocus,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Item>) {
  const row = useRow();
  return (
    <DropdownMenuPrimitive.Item
      {...props}
      className={cn(ROW, className)}
      data-fluid-hover-index={row["data-fluid-hover-index"]}
      onFocus={(event) => {
        row.onFocus();
        onFocus?.(event);
      }}
      ref={row.ref as React.Ref<HTMLDivElement>}
    />
  );
}

function DropdownMenuRadioItem({
  className,
  children,
  onFocus,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.RadioItem>) {
  const row = useRow();
  return (
    <DropdownMenuPrimitive.RadioItem
      {...props}
      className={cn(
        ROW,
        "data-[state=checked]:bg-accent-wash data-[state=checked]:text-accent",
        className,
      )}
      data-fluid-hover-index={row["data-fluid-hover-index"]}
      onFocus={(event) => {
        row.onFocus();
        onFocus?.(event);
      }}
      ref={row.ref as React.Ref<HTMLDivElement>}
    >
      {children}
      <DropdownMenuPrimitive.ItemIndicator className="ml-auto flex">
        <Check aria-hidden="true" className="stroke-2!" />
      </DropdownMenuPrimitive.ItemIndicator>
    </DropdownMenuPrimitive.RadioItem>
  );
}

function DropdownMenuLabel({
  className,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Label>) {
  return (
    <DropdownMenuPrimitive.Label
      className={cn("px-2 pt-1.5 pb-2 text-meta text-mute", className)}
      {...props}
    />
  );
}

function DropdownMenuSeparator({
  className,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Separator>) {
  return (
    <DropdownMenuPrimitive.Separator
      className={cn("-mx-1 my-1 h-px shrink-0 bg-rule/60", className)}
      {...props}
    />
  );
}

export {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
};
