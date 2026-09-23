"use client";

import { Pause, Play } from "lucide-react";
import { type ReactNode, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";

export function LandingMotion({ children }: { children: ReactNode }) {
  const root = useRef<HTMLDivElement>(null);
  const toggle = useRef<() => void>(() => {});
  const [mode, setMode] = useState("pending");

  useEffect(() => {
    const element = root.current;
    if (!element) return;
    let disposed = false;
    let dispose: (() => void) | undefined;
    import("./journey")
      .then(({ startJourney }) => {
        if (disposed) return;
        const journey = startJourney(element, setMode);
        toggle.current = journey.toggle;
        dispose = journey.dispose;
      })
      .catch(() => {
        if (!disposed) setMode("unavailable");
      });
    return () => {
      disposed = true;
      dispose?.();
    };
  }, []);

  const live = mode === "live" || mode === "loading";
  return (
    <div ref={root} className="illarin-journey" data-mode={mode}>
      {children}
      <Button
        variant="ghost"
        size="compact"
        className="motion-control"
        hidden={mode === "pending"}
        type="button"
        aria-pressed={live}
        disabled={mode === "reduced" || mode === "unavailable"}
        onClick={() => toggle.current()}
      >
        {live ? <Pause aria-hidden="true" /> : <Play aria-hidden="true" />}
        {mode === "reduced"
          ? "Reduced motion"
          : mode === "unavailable"
            ? "Still version"
            : live
              ? "Pause motion"
              : "Enable motion"}
      </Button>
      <output className="sr-only">
        {mode === "unavailable"
          ? "The animated artwork could not load. The still version is available."
          : ""}
      </output>
    </div>
  );
}
