"use client";

import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import { type ReactNode, useId } from "react";
import { cn, focusRing } from "@/lib/cn";

type CheckboxProps = {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  id?: string;
  name?: string;
  "aria-labelledby"?: string;
  "aria-describedby"?: string;
  "aria-invalid"?: boolean;
  ref?: React.Ref<HTMLButtonElement>;
};

/** Checkbox is the box alone: a grey fill that turns violet with a white tick. */
export function Checkbox({
  checked,
  onCheckedChange,
  ...props
}: CheckboxProps) {
  return (
    <CheckboxPrimitive.Root
      {...props}
      checked={checked}
      className={cn(
        "grid size-4 shrink-0 cursor-pointer place-items-center rounded-chip bg-off transition-colors duration-80 group-hover/row:bg-off-hover hover:bg-off-hover disabled:cursor-default disabled:opacity-50 aria-invalid:ring-1 aria-invalid:ring-stop data-[state=checked]:bg-action group-hover/row:data-[state=checked]:bg-action-hover",
        focusRing,
        "focus-visible:ring-offset-1 focus-visible:ring-offset-field",
      )}
      onCheckedChange={(next: CheckboxPrimitive.CheckedState) =>
        onCheckedChange(next === true)
      }
    >
      <CheckboxPrimitive.Indicator asChild>
        <svg
          aria-hidden="true"
          className="size-3.5 text-on-accent"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2.5}
          viewBox="0 0 24 24"
        >
          <path d="M5 12.5L10 17L19 7" />
        </svg>
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}

type RowProps = {
  htmlFor: string;
  label: ReactNode;
  hint?: ReactNode;
  hintId?: string;
  className?: string;
  children: ReactNode;
};

function Row({ htmlFor, label, hint, hintId, className, children }: RowProps) {
  return (
    <label
      htmlFor={htmlFor}
      className={cn(
        "group/row flex min-h-control cursor-pointer items-start gap-2.5 py-[calc((var(--control)-1lh)/2)] font-ui text-ui text-ink has-disabled:cursor-default has-disabled:opacity-50",
        className,
      )}
    >
      <span className="flex h-lh shrink-0 items-center">{children}</span>
      <span className="grid min-w-0 gap-0.5">
        <span>{label}</span>
        {hint ? (
          <span className="text-meta text-mute" id={hintId}>
            {hint}
          </span>
        ) : null}
      </span>
    </label>
  );
}

/** CheckboxRow is one checkbox with its label and an optional hint, for a lone yes-or-no inside a form. */
export function CheckboxRow({
  label,
  hint,
  className,
  ...box
}: CheckboxProps & {
  label: ReactNode;
  hint?: ReactNode;
  className?: string;
}) {
  const generated = useId();
  const id = box.id ?? generated;
  return (
    <Row className={className} hint={hint} htmlFor={id} label={label}>
      <Checkbox {...box} id={id} />
    </Row>
  );
}

type CheckboxOption<T extends string> = {
  value: T;
  label: ReactNode;
  hint?: ReactNode;
  disabled?: boolean;
};

/** CheckboxGroup is several checkbox rows, any of them ticked. */
export function CheckboxGroup<T extends string>({
  options,
  value,
  onValueChange,
  disabled,
  className,
}: {
  options: readonly CheckboxOption<T>[];
  value: readonly T[];
  onValueChange: (value: T[]) => void;
  disabled?: boolean;
  className?: string;
}) {
  const id = useId();
  return (
    <div className={cn("grid max-w-md min-w-0", className)}>
      {options.map((option, index) => {
        const checked = value.includes(option.value);
        const hintId = option.hint ? `${id}-${index}-hint` : undefined;
        return (
          <Row
            hint={option.hint}
            hintId={hintId}
            htmlFor={`${id}-${index}`}
            key={option.value}
            label={option.label}
          >
            <Checkbox
              aria-describedby={hintId}
              checked={checked}
              disabled={disabled || option.disabled}
              id={`${id}-${index}`}
              onCheckedChange={(on) =>
                onValueChange(
                  on
                    ? [...value, option.value]
                    : value.filter((one) => one !== option.value),
                )
              }
            />
          </Row>
        );
      })}
    </div>
  );
}
