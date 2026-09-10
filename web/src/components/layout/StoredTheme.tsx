"use client";

import { useEffect } from "react";
import { readThemePreference } from "@/lib/theme";

/** Applies the saved appearance on the pages the browser draws itself, such as a not-found or error answer */
export function StoredTheme() {
  useEffect(() => {
    const saved = readThemePreference();
    if (saved !== "system") document.documentElement.dataset.theme = saved;
  }, []);
  return null;
}
