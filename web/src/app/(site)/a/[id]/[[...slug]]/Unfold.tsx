"use client";

import { useReducedMotion } from "framer-motion";
import { ChevronDown, Maximize2 } from "lucide-react";
import { type ReactNode, useLayoutEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleTrigger } from "@/components/ui/collapsible";

/** Unfold opens an excerpt where it sits rather than in a second copy of it. */
export function Unfold({
  children,
  id,
  isCut,
  more,
  onToggle,
  open,
  panelId,
}: {
  children: ReactNode;
  id: string;
  isCut: boolean;
  more: string;
  onToggle: () => void;
  open: boolean;
  panelId: string;
}) {
  const shell = useRef<HTMLDivElement>(null);
  const region = useRef<HTMLDivElement>(null);
  const measured = useRef<number | null>(null);
  const still = useReducedMotion();

  // biome-ignore lint/correctness/useExhaustiveDependencies: Opening and closing is what asks for the new measurement.
  useLayoutEffect(() => {
    const box = shell.current;
    const inner = region.current;
    if (!box || !inner) return;

    const from = measured.current;
    const to = inner.getBoundingClientRect().height;
    measured.current = to;
    if (from === null || Math.abs(to - from) < 1 || still) return;

    const rest = () => {
      box.style.transition = "";
      box.style.height = "";
      box.style.overflow = "";
    };

    box.style.overflow = "hidden";
    box.style.transition = "";
    box.style.height = `${from}px`;
    void box.getBoundingClientRect().height;
    box.style.transition = "height 240ms var(--ease-wipe)";
    box.style.height = `${to}px`;
    box.addEventListener("transitionend", rest, { once: true });

    return () => {
      box.removeEventListener("transitionend", rest);
      rest();
    };
  }, [open, still]);

  return (
    <Collapsible
      className="flex min-w-0 flex-col"
      open={open}
      onOpenChange={onToggle}
    >
      <div className="min-w-0" ref={shell}>
        <div ref={region}>{children}</div>
      </div>
      {isCut || open ? (
        <CollapsibleTrigger asChild>
          <Button
            aria-controls={panelId}
            className="group/unfold mt-1 self-start"
            id={id}
          >
            <ChevronDown
              aria-hidden="true"
              className="transition-transform duration-160 group-data-[state=open]/unfold:rotate-180 motion-reduce:transition-none"
            />
            {open ? "Show less" : more}
          </Button>
        </CollapsibleTrigger>
      ) : null}
    </Collapsible>
  );
}

/** Browse shows a run's opening rows and hands the rest to a roomier surface. */
export function Browse({
  children,
  label,
  onOpen,
}: {
  children: ReactNode;
  label: string;
  onOpen: () => void;
}) {
  return (
    <>
      <div className="min-w-0">{children}</div>
      <Button className="mt-1 self-start" onClick={onOpen}>
        <Maximize2 aria-hidden="true" />
        {label}
      </Button>
    </>
  );
}
