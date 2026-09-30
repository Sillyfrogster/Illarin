"use client";

import * as RadioGroupPrimitive from "@radix-ui/react-radio-group";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import {
  type ChoiceGroupProps,
  useChoicePlates,
} from "@/components/ui/radio-group";
import { WeightLabel } from "@/components/ui/weight-label";
import { cn, focusRing } from "@/lib/cn";

/** Segmented picks one short value, drawn as Fluid Functionalism's tabs: the chosen segment's plate slides to it and a hover plate follows the pointer. */
export function Segmented<T extends string>({
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
    "x",
  );
  return (
    <RadioGroupPrimitive.Root
      {...labelled}
      className={cn(
        "relative inline-flex max-w-full items-center rounded-plate bg-deep p-1 select-none",
        className,
      )}
      disabled={disabled}
      name={name}
      onMouseEnter={hover.handlers.onMouseEnter}
      onMouseLeave={hover.handlers.onMouseLeave}
      onMouseMove={hover.handlers.onMouseMove}
      onValueChange={(next) => onValueChange(next as T)}
      orientation="horizontal"
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
              "relative z-10 inline-flex h-segment min-w-16 flex-1 cursor-pointer items-center justify-center rounded-control px-3 font-ui text-ui whitespace-nowrap transition-colors duration-80 data-[disabled]:cursor-default data-[disabled]:opacity-50",
              chosen ? "text-accent" : "text-mute hover:text-ink",
              focusRing,
            )}
            data-fluid-hover-index={index}
            disabled={option.disabled}
            key={option.value}
            onFocus={() => hover.setActiveIndex(index)}
            ref={itemRefs[index]}
            value={option.value}
          >
            <WeightLabel chosen={chosen}>{option.label}</WeightLabel>
          </RadioGroupPrimitive.Item>
        );
      })}
    </RadioGroupPrimitive.Root>
  );
}
