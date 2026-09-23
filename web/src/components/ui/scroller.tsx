"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { type ReactNode, useLayoutEffect, useRef, useState } from "react";
import { cn } from "@/lib/cn";

const FADE = "3.5rem";

const edge =
  "absolute z-10 grid size-9 place-items-center rounded-full bg-plane text-ink shadow-[0_4px_14px_-4px_rgb(0_0_0/0.35)] inset-ring inset-ring-rule/60 transition-[opacity,scale] duration-200 hover:scale-105 motion-reduce:transition-none";

/** Scroller keeps its children on one row, fading and offering a button at whichever edge hides more of them. */
export function Scroller({
  buttonClassName = "top-1/2 -translate-y-1/2",
  children,
  className,
}: {
  buttonClassName?: string;
  children: ReactNode;
  className?: string;
}) {
  const row = useRef<HTMLDivElement>(null);
  const [hidden, setHidden] = useState({ left: false, right: false });

  useLayoutEffect(() => {
    const here = row.current;
    if (!here) return;
    const measure = () => {
      const left = here.scrollLeft > 1;
      const right = here.scrollLeft + here.clientWidth < here.scrollWidth - 1;
      setHidden((was) =>
        was.left === left && was.right === right ? was : { left, right },
      );
    };
    measure();
    here.addEventListener("scroll", measure, { passive: true });
    const observer = new ResizeObserver(measure);
    observer.observe(here);
    for (const child of here.children) observer.observe(child);
    return () => {
      here.removeEventListener("scroll", measure);
      observer.disconnect();
    };
  }, []);

  function page(direction: 1 | -1) {
    const here = row.current;
    if (!here) return;
    const still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    here.scrollBy({
      left: direction * here.clientWidth * 0.7,
      behavior: still ? "auto" : "smooth",
    });
  }

  const mask = `linear-gradient(to right, ${hidden.left ? "transparent" : "#000"}, #000 ${FADE}, #000 calc(100% - ${FADE}), ${hidden.right ? "transparent" : "#000"})`;

  return (
    <div className="relative">
      <div
        className={cn(
          "overflow-x-auto overscroll-x-contain [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
          className,
        )}
        ref={row}
        style={{ maskImage: mask, WebkitMaskImage: mask }}
      >
        {children}
      </div>
      <button
        aria-hidden="true"
        className={cn(
          edge,
          buttonClassName,
          "left-0",
          !hidden.left && "pointer-events-none scale-90 opacity-0",
        )}
        onClick={() => page(-1)}
        tabIndex={-1}
        type="button"
      >
        <ChevronLeft className="size-4" />
      </button>
      <button
        aria-hidden="true"
        className={cn(
          edge,
          buttonClassName,
          "right-0",
          !hidden.right && "pointer-events-none scale-90 opacity-0",
        )}
        onClick={() => page(1)}
        tabIndex={-1}
        type="button"
      >
        <ChevronRight className="size-4" />
      </button>
    </div>
  );
}
