"use client";

import * as RadioGroupPrimitive from "@radix-ui/react-radio-group";
import { motion, useReducedMotion } from "framer-motion";
import { type ReactNode, useId, useMemo, useRef } from "react";
import {
  FluidHoverHighlight,
  toTarget,
} from "@/components/ui/fluid-hover-highlight";
import { WeightLabel } from "@/components/ui/weight-label";
import { cn, focusRing } from "@/lib/cn";
import { spring } from "@/lib/springs";
import { useFluidHover } from "@/lib/use-fluid-hover";

type Choice<T extends string> = {
  value: T;
  label: ReactNode;
  hint?: ReactNode;
  media?: ReactNode;
  disabled?: boolean;
};

type ChoiceGroupProps<T extends string> = {
  options: readonly Choice<T>[];
  value: T | null;
  onValueChange: (value: T) => void;
  disabled?: boolean;
  name?: string;
  className?: string;
  "aria-label"?: string;
  "aria-labelledby"?: string;
};

function isItemDisabled(element: HTMLElement) {
  return element.hasAttribute("data-disabled");
}

/** useChoicePlates measures a group's items for the hover plate and the plate under the chosen item. */
function useChoicePlates<T extends string>(
  options: readonly Choice<T>[],
  value: T | null,
  axis: "x" | "y",
) {
  const containerRef = useRef<HTMLDivElement>(null);
  const hover = useFluidHover(containerRef, { axis, isItemDisabled });
  const { registerItem } = hover;
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
  const reduceMotion = useReducedMotion();
  const chosen = options.findIndex((option) => option.value === value);
  const rect = hover.isMeasured ? hover.itemRects[chosen] : undefined;
  const chosenPlate = (className: string) =>
    rect ? (
      <motion.div
        animate={toTarget(rect)}
        className={cn("pointer-events-none absolute top-0 left-0", className)}
        initial={false}
        transition={reduceMotion ? { duration: 0 } : spring.moderate}
      />
    ) : null;
  return { containerRef, hover, itemRefs, chosenPlate };
}

/** RadioGroup is Fluid Functionalism's radio group: rows with an optional hint, a plate that follows the pointer, and the chosen row on the violet wash. */
function RadioGroup<T extends string>({
  options,
  value,
  onValueChange,
  disabled,
  name,
  className,
  ...labelled
}: ChoiceGroupProps<T>) {
  const { containerRef, hover, itemRefs, chosenPlate } = useChoicePlates(
    options,
    value,
    "y",
  );
  const id = useId();
  return (
    <RadioGroupPrimitive.Root
      {...labelled}
      className={cn(
        "relative -mx-3 grid w-[calc(100%+1.5rem)] max-w-[calc(24rem+1.5rem)] min-w-0 select-none",
        className,
      )}
      disabled={disabled}
      name={name}
      onMouseEnter={hover.handlers.onMouseEnter}
      onMouseLeave={hover.handlers.onMouseLeave}
      onMouseMove={hover.handlers.onMouseMove}
      onValueChange={(next) => onValueChange(next as T)}
      ref={containerRef}
      value={value ?? ""}
    >
      {chosenPlate("rounded-control bg-accent-wash")}
      <FluidHoverHighlight className="rounded-control" hover={hover} />
      {options.map((option, index) => {
        const chosen = option.value === value;
        return (
          <RadioGroupPrimitive.Item
            className={cn(
              "group/radio relative z-10 flex min-h-control cursor-pointer gap-2.5 rounded-control px-3 text-left font-ui text-ui text-ink data-[disabled]:cursor-default data-[disabled]:opacity-50",
              option.media
                ? "items-center py-0.5"
                : "items-start py-[calc((var(--control)-1lh)/2)]",
              focusRing,
            )}
            data-fluid-hover-index={index}
            disabled={option.disabled}
            key={option.value}
            onFocus={() => hover.setActiveIndex(index)}
            ref={itemRefs[index]}
            value={option.value}
          >
            <span className="flex h-lh shrink-0 items-center">
              <span
                className={cn(
                  "grid size-4 place-items-center rounded-full border-[1.5px] transition-colors duration-80",
                  chosen
                    ? "border-action"
                    : "border-edge group-hover/radio:border-mute",
                )}
              >
                <RadioGroupPrimitive.Indicator asChild>
                  <motion.span
                    animate={{ opacity: 1, scale: 1 }}
                    className="size-2 rounded-full bg-action"
                    initial={{ opacity: 0, scale: 0.3 }}
                    transition={spring.fast}
                  />
                </RadioGroupPrimitive.Indicator>
              </span>
            </span>
            {option.media}
            <span className="grid min-w-0 gap-0.5">
              <WeightLabel
                chosen={chosen}
                className={cn(
                  "transition-colors duration-80",
                  chosen && "text-accent",
                )}
                id={`${id}-${index}-label`}
              >
                {option.label}
              </WeightLabel>
              {option.hint ? (
                <span
                  className="text-meta text-mute"
                  id={`${id}-${index}-hint`}
                >
                  {option.hint}
                </span>
              ) : null}
            </span>
          </RadioGroupPrimitive.Item>
        );
      })}
    </RadioGroupPrimitive.Root>
  );
}

export { RadioGroup, useChoicePlates };
export type { Choice, ChoiceGroupProps };
