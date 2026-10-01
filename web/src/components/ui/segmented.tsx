"use client";

import * as RadioGroupPrimitive from "@radix-ui/react-radio-group";
import type { ChoiceGroupProps } from "@/components/ui/radio-group";
import { cn, focusRing } from "@/lib/cn";

/** Segmented picks one short value from a grey bar; the chosen segment fills violet. */
export function Segmented<T extends string>({
  options,
  value,
  onValueChange,
  disabled,
  name,
  className,
  ...labelled
}: ChoiceGroupProps<T>) {
  return (
    <RadioGroupPrimitive.Root
      {...labelled}
      className={cn(
        "inline-flex max-w-full items-center gap-0.5 rounded-control bg-fill p-1 select-none",
        className,
      )}
      disabled={disabled}
      name={name}
      onValueChange={(next) => onValueChange(next as T)}
      orientation="horizontal"
      value={value ?? ""}
    >
      {options.map((option) => (
        <RadioGroupPrimitive.Item
          className={cn(
            "inline-flex h-segment min-w-16 flex-1 cursor-pointer items-center justify-center rounded-chip px-3 font-ui text-ui font-medium whitespace-nowrap text-mute transition-colors duration-80 hover:text-ink data-disabled:cursor-default data-disabled:opacity-50 data-[state=checked]:bg-action data-[state=checked]:text-on-accent focus-visible:ring-offset-1 focus-visible:ring-offset-fill",
            focusRing,
          )}
          disabled={option.disabled}
          key={option.value}
          value={option.value}
        >
          {option.label}
        </RadioGroupPrimitive.Item>
      ))}
    </RadioGroupPrimitive.Root>
  );
}
