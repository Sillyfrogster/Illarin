"use client";

import * as SwitchPrimitive from "@radix-ui/react-switch";
import { type ReactNode, useId } from "react";
import { cn, focusRing } from "@/lib/cn";

type SwitchProps = {
  label: ReactNode;
  hint?: ReactNode;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  className?: string;
};

/** Switch is on or off the moment it is pressed: a grey track that turns violet as its white thumb slides across. */
function Switch({
  label,
  hint,
  checked,
  onCheckedChange,
  disabled = false,
  className,
}: SwitchProps) {
  const id = useId();
  return (
    <label
      htmlFor={id}
      className={cn(
        "group/switch flex min-h-control cursor-pointer items-start gap-2.5 py-[calc((var(--control)-1lh)/2)] font-ui text-ui text-ink select-none has-disabled:cursor-default has-disabled:opacity-50",
        className,
      )}
    >
      <span className="flex h-lh shrink-0 items-center">
        <SwitchPrimitive.Root
          aria-describedby={hint ? `${id}-hint` : undefined}
          checked={checked}
          className={cn(
            "flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full bg-off p-0.5 transition-colors duration-80 group-hover/switch:bg-off-hover disabled:cursor-default data-[state=checked]:bg-action group-hover/switch:data-[state=checked]:bg-action-hover",
            focusRing,
            "focus-visible:ring-offset-1 focus-visible:ring-offset-field",
          )}
          disabled={disabled}
          id={id}
          onCheckedChange={onCheckedChange}
        >
          <SwitchPrimitive.Thumb className="size-4 rounded-full bg-on-accent transition-transform duration-160 ease-(--ease-wipe) data-[state=checked]:translate-x-4" />
        </SwitchPrimitive.Root>
      </span>
      <span className="grid min-w-0 gap-0.5">
        <span>{label}</span>
        {hint ? (
          <span className="text-meta text-mute" id={`${id}-hint`}>
            {hint}
          </span>
        ) : null}
      </span>
    </label>
  );
}

export { Switch };
export type { SwitchProps };
