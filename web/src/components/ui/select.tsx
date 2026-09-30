"use client";

import * as SelectPrimitive from "@radix-ui/react-select";
import { motion } from "framer-motion";
import { Check, ChevronDown } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useFieldControl } from "@/components/ui/field";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import { inputClasses } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/cn";
import { Elevated } from "@/lib/elevated";
import {
  isDisabledRow,
  popupMotionClass,
  popupScrollAreaClass,
  popupViewportClass,
} from "@/lib/popup";
import { spring } from "@/lib/springs";
import { useFluidHover } from "@/lib/use-fluid-hover";
import { usePresence } from "@/lib/use-presence";

/** EMPTY carries an empty option through Radix, which reserves the empty string for "no value". */
const EMPTY = "\u0000empty";

type SelectOption<T extends string> = { value: T; label: ReactNode };

type SelectProps<T extends string> = {
  options: readonly SelectOption<T>[];
  value: T;
  onValueChange: (value: T) => void;
  disabled?: boolean;
  id?: string;
  className?: string;
  "aria-label"?: string;
  "aria-describedby"?: string;
};

/** Select is Fluid Functionalism's select: an outlined trigger that opens a raised list, a plate that follows the pointer and keys, and the chosen row violet with a check. */
export function Select<T extends string>({
  options,
  value,
  onValueChange,
  disabled,
  className,
  ...props
}: SelectProps<T>) {
  const field = useFieldControl(props);
  const [open, setOpen] = useState(false);
  const { mounted, onExitComplete } = usePresence(open, spring.fast);
  return (
    <SelectPrimitive.Root
      disabled={disabled}
      onOpenChange={setOpen}
      onValueChange={(next) => onValueChange((next === EMPTY ? "" : next) as T)}
      open={mounted}
      value={value === "" ? EMPTY : value}
    >
      <SelectPrimitive.Trigger
        {...props}
        {...field}
        className={cn(
          inputClasses,
          "group flex cursor-pointer items-center justify-between gap-2 text-left data-[state=open]:border-accent data-[placeholder]:text-mute",
          className,
        )}
      >
        <span className="min-w-0 flex-1 truncate">
          <SelectPrimitive.Value />
        </span>
        <SelectPrimitive.Icon asChild>
          <ChevronDown
            aria-hidden="true"
            className="size-4 shrink-0 text-mute transition-transform duration-160 group-data-[state=open]:rotate-180"
          />
        </SelectPrimitive.Icon>
      </SelectPrimitive.Trigger>
      <SelectList
        onExitComplete={onExitComplete}
        open={open}
        options={options}
        value={value}
      />
    </SelectPrimitive.Root>
  );
}

function SelectList<T extends string>({
  open,
  options,
  value,
  onExitComplete,
}: {
  open: boolean;
  options: readonly SelectOption<T>[];
  value: T;
  onExitComplete: () => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const hover = useFluidHover(containerRef, { isItemDisabled: isDisabledRow });
  const { registerItem, remeasure, setActiveIndex } = hover;
  const count = options.length;
  const itemRefs = useMemo(
    () =>
      Array.from(
        { length: count },
        (_, index) => (element: HTMLElement | null) =>
          registerItem(index, element),
      ),
    [count, registerItem],
  );

  useEffect(() => {
    if (open) remeasure();
  }, [open, remeasure]);

  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Content
        align="start"
        className="z-90"
        collisionPadding={16}
        position="popper"
        sideOffset={6}
      >
        <motion.div
          animate={
            open
              ? { opacity: 1, y: 0, scaleY: 1 }
              : { opacity: 0, y: "var(--popup-enter-y)", scaleY: 0.96 }
          }
          className={popupMotionClass}
          initial={{ opacity: 0, y: "var(--popup-enter-y)", scaleY: 0.96 }}
          onAnimationComplete={onExitComplete}
          transition={open ? spring.fast : spring.fast.exit}
        >
          <SelectPrimitive.Viewport asChild>
            <Elevated
              className="flex max-h-[min(20rem,var(--radix-select-content-available-height))] min-w-(--radix-select-trigger-width) max-w-[calc(100vw-2rem)] flex-col overflow-hidden rounded-plate font-ui text-ink outline-none select-none"
              offset={2}
              onMouseEnter={hover.handlers.onMouseEnter}
              onMouseLeave={hover.handlers.onMouseLeave}
              onMouseMove={hover.handlers.onMouseMove}
              shadowLevel={3}
            >
              <ScrollArea
                className={popupScrollAreaClass}
                viewportClassName={cn(popupViewportClass, "scroll-fade")}
              >
                <div className="relative flex flex-col p-1" ref={containerRef}>
                  <FluidHoverHighlight
                    className="rounded-control"
                    hidden={!open}
                    hover={hover}
                  />
                  {options.map((option, index) => (
                    <SelectPrimitive.Item
                      className="relative z-10 flex min-h-control shrink-0 cursor-pointer items-center gap-2 rounded-control px-2 text-ui text-mute outline-none transition-colors duration-80 data-[highlighted]:text-ink data-[state=checked]:bg-accent-wash data-[state=checked]:text-accent data-[disabled]:pointer-events-none data-[disabled]:opacity-50"
                      data-fluid-hover-index={index}
                      key={option.value}
                      onFocus={() => setActiveIndex(index)}
                      ref={itemRefs[index]}
                      value={option.value === "" ? EMPTY : option.value}
                    >
                      <span className="min-w-0 flex-1 truncate">
                        <SelectPrimitive.ItemText>
                          {option.label}
                        </SelectPrimitive.ItemText>
                      </span>
                      <span aria-hidden="true" className="size-4 shrink-0">
                        {option.value === value ? (
                          <Check className="size-4" />
                        ) : null}
                      </span>
                    </SelectPrimitive.Item>
                  ))}
                </div>
              </ScrollArea>
            </Elevated>
          </SelectPrimitive.Viewport>
        </motion.div>
      </SelectPrimitive.Content>
    </SelectPrimitive.Portal>
  );
}
