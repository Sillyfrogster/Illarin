"use client";

import * as SelectPrimitive from "@radix-ui/react-select";
import { Check, ChevronDown } from "lucide-react";
import type { ReactNode } from "react";
import { useFieldControl } from "@/components/ui/field";
import { inputClasses } from "@/components/ui/input";
import { cn, popupRow, popupSurface } from "@/lib/cn";

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

/** Select is a filled field that opens a list under it, the chosen row violet with a check. */
export function Select<T extends string>({
  options,
  value,
  onValueChange,
  disabled,
  className,
  ...props
}: SelectProps<T>) {
  const field = useFieldControl(props);
  return (
    <SelectPrimitive.Root
      disabled={disabled}
      onValueChange={(next) => onValueChange((next === EMPTY ? "" : next) as T)}
      value={value === "" ? EMPTY : value}
    >
      <SelectPrimitive.Trigger
        {...props}
        {...field}
        className={cn(
          inputClasses,
          "group flex cursor-pointer items-center justify-between gap-2 text-left data-[state=open]:bg-fill-hover data-placeholder:text-mute",
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
      <SelectPrimitive.Portal>
        <SelectPrimitive.Content
          align="start"
          className={cn(
            popupSurface,
            "max-h-(--radix-select-content-available-height) min-w-(--radix-select-trigger-width) max-w-[calc(100vw-2rem)] overflow-hidden select-none",
          )}
          collisionPadding={16}
          position="popper"
          sideOffset={6}
        >
          <SelectPrimitive.Viewport className="max-h-80 p-1">
            {options.map((option) => (
              <SelectPrimitive.Item
                className={cn(popupRow, "data-[state=checked]:text-accent")}
                key={option.value}
                value={option.value === "" ? EMPTY : option.value}
              >
                <span className="min-w-0 flex-1 truncate">
                  <SelectPrimitive.ItemText>
                    {option.label}
                  </SelectPrimitive.ItemText>
                </span>
                <SelectPrimitive.ItemIndicator>
                  <Check aria-hidden="true" />
                </SelectPrimitive.ItemIndicator>
              </SelectPrimitive.Item>
            ))}
          </SelectPrimitive.Viewport>
        </SelectPrimitive.Content>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
}
