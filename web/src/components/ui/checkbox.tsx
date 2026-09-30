"use client";

import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import { motion } from "framer-motion";
import { type ReactNode, useId, useMemo, useRef } from "react";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import { WeightLabel } from "@/components/ui/weight-label";
import { cn, focusRing } from "@/lib/cn";
import { spring } from "@/lib/springs";
import { useFluidHover } from "@/lib/use-fluid-hover";

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

/** Checkbox is the box alone: violet when ticked, with the tick drawn on the fast spring. */
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
        "peer grid size-4 shrink-0 cursor-pointer place-items-center rounded-[5px] border-[1.5px] border-edge transition-colors duration-80 hover:border-mute disabled:cursor-default disabled:opacity-50 aria-invalid:border-stop data-[state=checked]:border-action data-[state=checked]:bg-action",
        focusRing,
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
          <motion.path
            animate={{ pathLength: 1 }}
            d="M5 12.5L10 17L19 7"
            initial={{ pathLength: 0 }}
            transition={spring.fast}
          />
        </svg>
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}

type RowProps = {
  htmlFor: string;
  label: ReactNode;
  hint?: ReactNode;
  checked?: boolean;
  className?: string;
  children: ReactNode;
};

function Row({ htmlFor, label, hint, checked, className, children }: RowProps) {
  return (
    <label
      htmlFor={htmlFor}
      className={cn(
        "relative z-10 flex min-h-control cursor-pointer items-start gap-2.5 py-[calc((var(--control)-1lh)/2)] font-ui text-ui text-ink has-disabled:cursor-default has-disabled:opacity-50",
        className,
      )}
    >
      <span className="flex h-lh shrink-0 items-center">{children}</span>
      <span className="grid min-w-0 gap-0.5">
        {checked === undefined ? (
          <span>{label}</span>
        ) : (
          <WeightLabel chosen={checked}>{label}</WeightLabel>
        )}
        {hint ? <span className="text-meta text-mute">{hint}</span> : null}
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

/** CheckboxGroup is Fluid Functionalism's checkbox group: several rows, any of them ticked, with a plate that follows the pointer. */
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
  const containerRef = useRef<HTMLDivElement>(null);
  const hover = useFluidHover(containerRef);
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
  const id = useId();
  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: the pointer only moves the hover plate; each row is its own checkbox
    <div
      className={cn(
        "relative -mx-3 grid w-[calc(100%+1.5rem)] max-w-[calc(28rem+1.5rem)] min-w-0",
        className,
      )}
      onMouseEnter={hover.handlers.onMouseEnter}
      onMouseLeave={hover.handlers.onMouseLeave}
      onMouseMove={hover.handlers.onMouseMove}
      ref={containerRef}
    >
      <FluidHoverHighlight className="rounded-control" hover={hover} />
      {options.map((option, index) => {
        const checked = value.includes(option.value);
        return (
          <div
            data-fluid-hover-index={index}
            key={option.value}
            ref={itemRefs[index]}
          >
            <Row
              checked={checked}
              className="px-3"
              htmlFor={`${id}-${index}`}
              hint={
                option.hint ? (
                  <span id={`${id}-${index}-hint`}>{option.hint}</span>
                ) : undefined
              }
              label={option.label}
            >
              <Checkbox
                aria-describedby={
                  option.hint ? `${id}-${index}-hint` : undefined
                }
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
          </div>
        );
      })}
    </div>
  );
}
