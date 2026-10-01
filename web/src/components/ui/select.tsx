"use client";

import * as SelectPrimitive from "@radix-ui/react-select";
import { Check, ChevronDown } from "lucide-react";
import type { ReactNode } from "react";
import { useFieldControl } from "@/components/ui/field";
import { inputClasses } from "@/components/ui/input";
import { cn } from "@/lib/cn";

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
          className="z-90 max-h-(--radix-select-content-available-height) min-w-(--radix-select-trigger-width) max-w-[calc(100vw-2rem)] origin-(--radix-select-content-transform-origin) animate-pop overflow-hidden rounded-art bg-plane font-ui text-ink shadow-popover ring-1 ring-ink/8 select-none"
          collisionPadding={16}
          position="popper"
          sideOffset={6}
        >
          <SelectPrimitive.Viewport className="max-h-80 p-1">
            {options.map((option) => (
              <SelectPrimitive.Item
                className="flex min-h-control cursor-pointer items-center gap-2 rounded-control px-2 text-ui outline-none focus-visible:outline-none data-highlighted:bg-fill-hover data-disabled:pointer-events-none data-disabled:opacity-50 data-[state=checked]:text-accent"
                key={option.value}
                value={option.value === "" ? EMPTY : option.value}
              >
                <span className="min-w-0 flex-1 truncate">
                  <SelectPrimitive.ItemText>
                    {option.label}
                  </SelectPrimitive.ItemText>
                </span>
                <SelectPrimitive.ItemIndicator>
                  <Check aria-hidden="true" className="size-4" />
                </SelectPrimitive.ItemIndicator>
              </SelectPrimitive.Item>
            ))}
          </SelectPrimitive.Viewport>
        </SelectPrimitive.Content>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
}
