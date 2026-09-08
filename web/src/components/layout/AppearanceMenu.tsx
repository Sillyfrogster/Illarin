"use client";

import { Monitor, Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  applyThemePreference,
  readThemePreference,
  type ThemePreference,
} from "@/lib/theme";

const THEME_CHANGE_EVENT = "illarin:theme-preference";

const CHOICES = [
  { value: "system", label: "Match the system", Icon: Monitor },
  { value: "light", label: "Light", Icon: Sun },
  { value: "dark", label: "Dark", Icon: Moon },
] as const;

/** The appearance choice, in one control that keeps all three states */
export function AppearanceMenu({
  className,
  labelled = false,
  embedded = false,
}: {
  className?: string;
  labelled?: boolean;
  embedded?: boolean;
}) {
  const [preference, setPreference] = useState<ThemePreference>("system");

  useEffect(() => {
    function syncPreference(event?: Event) {
      if (event instanceof CustomEvent && event.detail) {
        setPreference(event.detail as ThemePreference);
        return;
      }
      setPreference(readThemePreference());
    }

    syncPreference();
    window.addEventListener("storage", syncPreference);
    window.addEventListener(THEME_CHANGE_EVENT, syncPreference);
    return () => {
      window.removeEventListener("storage", syncPreference);
      window.removeEventListener(THEME_CHANGE_EVENT, syncPreference);
    };
  }, []);

  function handleChange(value: string) {
    const next = value as ThemePreference;
    setPreference(next);
    applyThemePreference(next);
    window.dispatchEvent(
      new CustomEvent<ThemePreference>(THEME_CHANGE_EVENT, { detail: next }),
    );
  }

  const current = CHOICES.find((choice) => choice.value === preference);
  const CurrentIcon = current?.Icon ?? Monitor;

  if (embedded) {
    return (
      <>
        <DropdownMenuLabel className="pt-1 pb-2 text-meta text-mute">
          Appearance
        </DropdownMenuLabel>
        <DropdownMenuRadioGroup
          aria-label="Appearance"
          value={preference}
          onValueChange={handleChange}
          className="mx-1 mb-1 grid grid-cols-3 gap-1 rounded-control bg-deep p-1"
        >
          {CHOICES.map(({ value, label, Icon }) => (
            <DropdownMenuRadioItem
              key={value}
              value={value}
              aria-label={label}
              onSelect={(event) => event.preventDefault()}
              className="min-h-14 flex-col justify-center gap-1 px-2 py-2 text-meta data-[highlighted]:ring-2 data-[highlighted]:ring-accent data-[state=checked]:bg-plane data-[state=checked]:text-ink [&>span]:hidden"
            >
              <Icon aria-hidden="true" />
              {value === "system" ? "System" : label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size={labelled ? "compact" : "icon"}
          className={className}
          aria-label={`Appearance: ${current?.label ?? "Match the system"}`}
        >
          <CurrentIcon />
          {labelled ? <span aria-hidden="true">Appearance</span> : null}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-56">
        <DropdownMenuLabel className="text-meta text-mute">
          Appearance
        </DropdownMenuLabel>
        <DropdownMenuRadioGroup value={preference} onValueChange={handleChange}>
          {CHOICES.map(({ value, label, Icon }) => (
            <DropdownMenuRadioItem key={value} value={value}>
              <Icon aria-hidden="true" />
              {label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
