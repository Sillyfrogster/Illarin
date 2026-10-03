"use client";

import * as RadioGroupPrimitive from "@radix-ui/react-radio-group";
import { type ReactNode, useId } from "react";
import { cn, focusRing } from "@/lib/cn";

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

/** RadioGroup is rows with an optional hint; each row's dot is a grey fill that turns violet with a white centre when chosen. */
function RadioGroup<T extends string>({
  options,
  value,
  onValueChange,
  disabled,
  name,
  className,
  ...labelled
}: ChoiceGroupProps<T>) {
  const id = useId();
  return (
    <RadioGroupPrimitive.Root
      {...labelled}
      className={cn("grid max-w-sm min-w-0 select-none", className)}
      disabled={disabled}
      name={name}
      onValueChange={(next) => onValueChange(next as T)}
      value={value ?? ""}
    >
      {options.map((option, index) => (
        <RadioGroupPrimitive.Item
          aria-describedby={option.hint ? `${id}-${index}-hint` : undefined}
          className={cn(
            "group/radio flex min-h-control cursor-pointer gap-2.5 rounded-control text-left font-ui text-ui text-ink data-disabled:cursor-default data-disabled:opacity-50",
            option.media
              ? "items-center py-0.5"
              : "items-start py-[calc((var(--control)-1lh)/2)]",
            focusRing,
          )}
          disabled={option.disabled}
          key={option.value}
          value={option.value}
        >
          <span className="flex h-lh shrink-0 items-center">
            <span className="grid size-4 place-items-center rounded-full bg-off transition-colors duration-80 group-hover/radio:bg-off-hover group-data-[state=checked]/radio:bg-action">
              <RadioGroupPrimitive.Indicator className="size-1.5 rounded-full bg-on-accent" />
            </span>
          </span>
          {option.media}
          <span className="grid min-w-0 gap-0.5">
            <span>{option.label}</span>
            {option.hint ? (
              <span className="text-meta text-mute" id={`${id}-${index}-hint`}>
                {option.hint}
              </span>
            ) : null}
          </span>
        </RadioGroupPrimitive.Item>
      ))}
    </RadioGroupPrimitive.Root>
  );
}

export { RadioGroup };
export type { Choice, ChoiceGroupProps };
