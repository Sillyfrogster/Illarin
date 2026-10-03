"use client";

import { RadioGroup } from "@/components/ui/radio-group";
import { Segmented } from "@/components/ui/segmented";
import type { AppName, NsfwPreference } from "@/lib/api/query";

export const ANY_APP = "any";

export const ADULT_CHOICES: {
  value: NsfwPreference;
  label: string;
  note: string;
}[] = [
  { value: "hidden", label: "Hide", note: "Adult work stays out of results." },
  { value: "blurred", label: "Blur", note: "Blur adult covers." },
  { value: "shown", label: "Show", note: "Show adult covers." },
];

/** AppChoice offers the registry's apps and any app, one of which is the reader's. */
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
  const choices = [...apps, { id: ANY_APP, label: "Any app" }].map((app) => ({
    value: app.id,
    label: app.label,
  }));
  return (
    <RadioGroup
      disabled={disabled}
      name={name}
      onValueChange={onChange}
      options={choices}
      value={value}
    />
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
    <Segmented
      disabled={disabled}
      name={name}
      onValueChange={onChange}
      options={ADULT_CHOICES}
      value={value}
    />
  );
}

export function adultNote(value: NsfwPreference): string {
  return ADULT_CHOICES.find((one) => one.value === value)?.note ?? "";
}
