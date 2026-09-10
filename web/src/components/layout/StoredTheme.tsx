"use client";

import { useEffect } from "react";
import { readThemePreference } from "@/lib/theme";

export function StoredTheme() {
  useEffect(() => {
    const saved = readThemePreference();
    if (saved !== "system") document.documentElement.dataset.theme = saved;
  }, []);
  return null;
}
