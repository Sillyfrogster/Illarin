"use client";

import type { ComponentProps } from "react";
import { useFieldControl } from "@/components/ui/field";
import { cn, focusRing } from "@/lib/cn";

/** inputClasses is the one outlined text field: neutral at rest, violet on hover and focus. */
export const inputClasses = `h-control w-full min-w-0 rounded-control border border-edge bg-transparent px-3 font-ui text-ui text-ink shadow-[0_1px_2px_0_rgb(0_0_0/0.05)] transition-[background-color,border-color,box-shadow] duration-80 placeholder:text-mute hover:border-accent/50 hover:bg-hover focus-visible:border-accent focus-visible:bg-transparent disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-stop aria-invalid:focus-visible:ring-stop/40 dark:bg-edge/10 ${focusRing}`;

/** Input is shadcn's input in the site's outlined style; inside a Field it takes the field's id, hint and error. */
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
