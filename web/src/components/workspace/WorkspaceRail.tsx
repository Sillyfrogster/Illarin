"use client";

import { motion, useReducedMotion } from "framer-motion";
import { ChevronLeft, X } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";
import { timing } from "@/lib/timing";

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

  return (
    <motion.aside
      animate={reduced ? { opacity: 1 } : { x: 0 }}
      aria-label={title}
      className={cn(
        "fixed inset-x-0 bottom-0 z-40 flex max-h-[76dvh] flex-col rounded-t-plate bg-plane shadow-popover",
        "lg:top-(--site-header-offset) lg:right-0 lg:transition-[top] lg:duration-240 lg:bottom-0 lg:left-auto lg:max-h-none lg:w-[28rem] lg:rounded-none",
        "before:absolute before:inset-x-0 before:top-0 before:h-0.5 before:content-['']",
        "lg:before:inset-y-0 lg:before:right-auto lg:before:left-0 lg:before:h-auto lg:before:w-0.5",
        tone === "stop" ? "before:bg-stop" : "before:bg-accent",
      )}
      exit={reduced ? { opacity: 0 } : { x: "100%" }}
      initial={reduced ? { opacity: 0 } : { x: "100%" }}
      transition={timing.settle}
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
          <Button
            aria-label={`Close ${title.toLowerCase()}`}
            className="shrink-0"
            onClick={onClose}
            size="icon"
            variant="ghost"
          >
            <X aria-hidden="true" />
          </Button>
        ) : null}
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-6 pt-2 pb-8 [container-name:rail] [container-type:inline-size] md:px-8 lg:pb-32">
        {children}
      </div>
    </motion.aside>
  );
}

export function RailBack({
  children,
  onClick,
}: {
  children: ReactNode;
  onClick: () => void;
}) {
  return (
    <Button
      className="-ml-3 self-start"
      onClick={onClick}
      size="compact"
      variant="ghost"
    >
      <ChevronLeft aria-hidden="true" />
      {children}
    </Button>
  );
}
