"use client";

import { type FocusEvent, type KeyboardEvent, useEffect, useRef } from "react";

const MOVES = ["ArrowLeft", "ArrowRight", "Home", "End"];

export function useToolbarKeys() {
  const bar = useRef<HTMLDivElement>(null);
  const at = useRef(0);

  function controls(): HTMLButtonElement[] {
    return [...(bar.current?.querySelectorAll("button") ?? [])];
  }

  function settle() {
    const found = controls();
    const reachable = found.filter((one) => !one.disabled);
    if (reachable.length === 0) return;
    const wanted = found[at.current];
    const stop = wanted && !wanted.disabled ? wanted : reachable[0];
    for (const one of found) one.tabIndex = one === stop ? 0 : -1;
  }

  useEffect(settle);

  function onFocus(event: FocusEvent<HTMLDivElement>) {
    const index = controls().indexOf(
      event.target as unknown as HTMLButtonElement,
    );
    if (index >= 0) at.current = index;
  }

  function onKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (!MOVES.includes(event.key)) return;
    const reachable = controls().filter((one) => !one.disabled);
    const from = reachable.indexOf(
      document.activeElement as unknown as HTMLButtonElement,
    );
    if (from < 0 || reachable.length === 0) return;
    event.preventDefault();
    const step = event.key === "ArrowRight" ? 1 : -1;
    const to =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? reachable.length - 1
          : (from + step + reachable.length) % reachable.length;
    reachable[to].focus();
  }

  return { bar, onFocus, onKeyDown };
}
