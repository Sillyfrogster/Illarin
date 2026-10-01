"use client";

import * as DropdownMenuPrimitive from "@radix-ui/react-dropdown-menu";
import { Monitor, Moon, Sun } from "lucide-react";
import { useId, useState } from "react";
import { cn, focusRing } from "@/lib/cn";
import {
  applyThemePreference,
  readThemePreference,
  type ThemePreference,
} from "@/lib/theme";

const CHOICES = [
  { value: "system", label: "System", Icon: Monitor },
  { value: "light", label: "Light", Icon: Sun },
  { value: "dark", label: "Dark", Icon: Moon },
] as const;

/** AppearanceChoices is the account menu's row for light, dark or the system's theme; picking one keeps the menu open. */
export function AppearanceChoices() {
  const [preference, setPreference] =
    useState<ThemePreference>(readThemePreference);
  const labelId = useId();

  return (
    <div className="flex min-h-control items-center gap-3 pr-1 pl-2">
      <span className="flex-1 text-ui" id={labelId}>
        Appearance
      </span>
      <DropdownMenuPrimitive.RadioGroup
        aria-labelledby={labelId}
        className="flex gap-0.5 rounded-control bg-fill p-0.5"
        onValueChange={(value) => {
          const next = value as ThemePreference;
          setPreference(next);
          applyThemePreference(next);
        }}
        value={preference}
      >
        {CHOICES.map(({ value, label, Icon }) => (
          <DropdownMenuPrimitive.RadioItem
            aria-label={label}
            className={cn(
              "grid h-segment-compact w-9 cursor-pointer place-items-center rounded-chip text-mute transition-colors duration-80 outline-none data-highlighted:text-ink data-[state=checked]:bg-action data-[state=checked]:text-on-accent [&_svg]:size-4",
              focusRing,
            )}
            key={value}
            onSelect={(event) => event.preventDefault()}
            value={value}
          >
            <Icon aria-hidden="true" />
          </DropdownMenuPrimitive.RadioItem>
        ))}
      </DropdownMenuPrimitive.RadioGroup>
    </div>
  );
}
