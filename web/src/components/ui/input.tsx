"use client";

import type { ComponentProps } from "react";
import { useFieldControl } from "@/components/ui/field";
import { cn } from "@/lib/cn";

/** fieldSurface is the filled ground every text field shares: grey, a step darker or lighter on hover, a violet border only on focus. */
export const fieldSurface =
  "rounded-control border border-transparent bg-fill font-ui text-ui text-ink transition-colors duration-80 hover:bg-fill-hover";

/** inputClasses is the one text field. */
export const inputClasses = `${fieldSurface} h-control w-full min-w-0 px-3 outline-none placeholder:text-mute focus-visible:border-accent focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-stop`;

/** Input is the one text field; inside a Field it takes the field's id, hint and error. */
export function Input({ className, ...props }: ComponentProps<"input">) {
  const field = useFieldControl(props);
  return (
    <input className={cn(inputClasses, className)} {...props} {...field} />
  );
}

/** Textarea is the multi-line Input. */
export function Textarea({ className, ...props }: ComponentProps<"textarea">) {
  const field = useFieldControl(props);
  return (
    <textarea
      className={cn(
        inputClasses,
        "h-auto min-h-24 resize-y py-2 leading-relaxed",
        className,
      )}
      {...props}
      {...field}
    />
  );
}
