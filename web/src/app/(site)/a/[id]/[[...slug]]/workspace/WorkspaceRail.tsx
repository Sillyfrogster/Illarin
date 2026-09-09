"use client";

import { motion, useReducedMotion } from "framer-motion";
import { X } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import { cn } from "@/lib/cn";

/** Work that needs its own space sits beside the page, so the page stays visible and writable. */
export function WorkspaceRail({
  children,
  description,
  onClose,
  title,
  tone = "accent",
}: {
  children: ReactNode;
  description?: string;
  onClose?: () => void;
  title: string;
  tone?: "accent" | "stop";
}) {
  const reduced = useReducedMotion();
  const heading = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    heading.current?.focus({ preventScroll: true });
  }, []);

  useEffect(() => {
    if (!onClose) return;
    function leave(event: KeyboardEvent) {
      // Escape belongs to whatever field is open before it belongs to the rail.
      const inField =
        event.target instanceof HTMLElement &&
        ["INPUT", "SELECT", "TEXTAREA"].includes(event.target.tagName);
      if (event.key === "Escape" && !inField) onClose?.();
    }
    document.addEventListener("keydown", leave);
    return () => document.removeEventListener("keydown", leave);
  }, [onClose]);

  return (
    <motion.aside
      animate={reduced ? { opacity: 1 } : { x: 0 }}
      aria-label={title}
      className={cn(
        "fixed inset-x-0 bottom-0 z-40 flex max-h-[76dvh] flex-col rounded-t-plate bg-plane shadow-popover",
        "lg:top-[var(--header-height)] lg:right-0 lg:bottom-0 lg:left-auto lg:max-h-none lg:w-[28rem] lg:rounded-none",
        "before:absolute before:inset-x-0 before:top-0 before:h-0.5 before:content-['']",
        "lg:before:inset-y-0 lg:before:right-auto lg:before:left-0 lg:before:h-auto lg:before:w-0.5",
        tone === "stop" ? "before:bg-stop" : "before:bg-accent",
      )}
      exit={reduced ? { opacity: 0 } : { x: "100%" }}
      initial={reduced ? { opacity: 0 } : { x: "100%" }}
      transition={{ duration: reduced ? 0 : 0.44, ease: [0.22, 1, 0.36, 1] }}
    >
      <div className="flex items-start justify-between gap-4 px-6 pt-7 pb-4 md:px-8">
        <div className="min-w-0">
          <h2
            className="font-display text-section font-medium text-ink outline-offset-3 wrap-anywhere"
            ref={heading}
            tabIndex={-1}
          >
            {title}
          </h2>
          {description ? (
            <p className="mt-2 text-meta text-mute">{description}</p>
          ) : null}
        </div>
        {onClose ? (
          <button
            aria-label={`Close ${title.toLowerCase()}`}
            className="inline-flex size-11 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-deep hover:text-ink"
            onClick={onClose}
            type="button"
          >
            <X aria-hidden="true" size={18} />
          </button>
        ) : null}
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-6 pt-2 pb-8 [container-name:rail] [container-type:inline-size] md:px-8 lg:pb-32">
        {children}
      </div>
    </motion.aside>
  );
}
