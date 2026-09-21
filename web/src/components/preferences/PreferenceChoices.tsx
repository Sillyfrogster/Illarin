"use client";

import { Check } from "lucide-react";
import type { AppName, NsfwPreference } from "@/lib/api/query";
import { cn } from "@/lib/cn";

export const ANY_APP = "any";

export const ADULT_CHOICES: {
  value: NsfwPreference;
  label: string;
  note: string;
}[] = [
  { value: "hidden", label: "Hide", note: "Adult work stays out of results." },
  { value: "blurred", label: "Blur", note: "Blur adult cover images." },
  { value: "shown", label: "Show", note: "Show adult cover images." },
];

const tile =
  "relative flex min-h-11 cursor-pointer items-center justify-between gap-2 rounded-control px-3.5 font-ui text-ui font-medium transition-colors duration-200 has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-accent has-disabled:cursor-default has-disabled:opacity-55 motion-reduce:transition-none";

/** AppChoice offers the registry's apps and any app as tiles, one of which is the reader's. */
export function AppChoice({
  apps,
  disabled,
  name,
  onChange,
  value,
}: {
  apps: AppName[];
  disabled?: boolean;
  name: string;
  onChange: (app: string) => void;
  value: string | null;
}) {
  const choices = [...apps, { id: ANY_APP, label: "Any app" }];

  return (
    <div className="grid grid-cols-2 gap-2">
      {choices.map((app) => {
        const chosen = value === app.id;
        return (
          <label
            className={cn(
              tile,
              chosen
                ? "bg-accent-wash text-ink inset-ring-2 inset-ring-action"
                : "bg-deep text-ink hover:bg-rule/45",
            )}
            key={app.id}
          >
            <input
              checked={chosen}
              className="sr-only"
              disabled={disabled}
              name={name}
              onChange={() => onChange(app.id)}
              type="radio"
              value={app.id}
            />
            <span className="min-w-0 truncate">{app.label}</span>
            <Check
              aria-hidden="true"
              className={cn(
                "size-4 shrink-0 text-accent transition duration-200 motion-reduce:transition-none",
                chosen ? "scale-100 opacity-100" : "scale-50 opacity-0",
              )}
            />
          </label>
        );
      })}
    </div>
  );
}

/** AdultContentChoice picks hide, blur or show as one segmented control. */
export function AdultContentChoice({
  disabled,
  name,
  onChange,
  value,
}: {
  disabled?: boolean;
  name: string;
  onChange: (preference: NsfwPreference) => void;
  value: NsfwPreference;
}) {
  return (
    <div className="inline-flex gap-1 rounded-control bg-deep p-1">
      {ADULT_CHOICES.map((option) => {
        const chosen = value === option.value;
        return (
          <label
            className={cn(
              "flex min-h-9 min-w-16 cursor-pointer items-center justify-center rounded-[7px] px-4 font-ui text-ui font-medium transition-colors duration-200 has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-accent has-disabled:cursor-default has-disabled:opacity-55 motion-reduce:transition-none",
              chosen
                ? "bg-action text-on-accent shadow-[0_2px_8px_-3px_var(--v-action)]"
                : "text-mute hover:text-ink",
            )}
            key={option.value}
          >
            <input
              checked={chosen}
              className="sr-only"
              disabled={disabled}
              name={name}
              onChange={() => onChange(option.value)}
              type="radio"
              value={option.value}
            />
            {option.label}
          </label>
        );
      })}
    </div>
  );
}

export function adultNote(value: NsfwPreference): string {
  return ADULT_CHOICES.find((one) => one.value === value)?.note ?? "";
}
