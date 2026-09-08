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
}: {
  className?: string;
  labelled?: boolean;
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
