"use client";

import { type KeyboardEvent, useCallback, useEffect, useRef } from "react";
import { POPUP_NAV_KEYS } from "@/lib/popup";

/** useKeyboardNavGate shows a popup's focus ring only after the reader moves through it with the keyboard. */
export function useKeyboardNavGate(open: boolean) {
  const keyboardNavRef = useRef(false);

  useEffect(() => {
    if (!open) return;

    const active = document.activeElement;
    keyboardNavRef.current =
      active instanceof HTMLElement && active.matches(":focus-visible");
  }, [open]);

  const trackKeyboardNav = useCallback((e: KeyboardEvent) => {
    if (POPUP_NAV_KEYS.includes(e.key)) keyboardNavRef.current = true;
  }, []);

  return { keyboardNavRef, trackKeyboardNav };
}
