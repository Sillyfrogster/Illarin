"use client";

import type { ReactNode } from "react";

/** CheckRow is a plain checkbox with its label, for a yes-or-no choice inside a form. */
export function CheckRow({
  checked,
  children,
  disabled,
  onChange,
}: {
  checked: boolean;
  children: ReactNode;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex min-h-11 cursor-pointer items-center gap-3 font-ui text-ui text-ink has-disabled:cursor-default has-disabled:opacity-60">
      <input
        checked={checked}
        className="size-4 shrink-0 accent-[var(--v-action)]"
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      {children}
    </label>
  );
}
