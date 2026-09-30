"use client";

import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/cn";

const LABEL = "shrink-0 font-ui text-meta text-mute";

export function Row({
  children,
  label,
  onClose,
}: {
  children: ReactNode;
  label: string;
  onClose?: () => void;
}) {
  return (
    <fieldset
      aria-label={label}
      className="flex flex-wrap items-center gap-2 border-t border-rule/60 bg-deep px-2.5 py-2.5"
      onKeyDown={(event) => {
        if (event.key === "Escape" && onClose) onClose();
      }}
    >
      <span aria-hidden="true" className={cn(LABEL, "px-1 font-medium")}>
        {label}
      </span>
      {children}
    </fieldset>
  );
}

export function Pair({
  children,
  field,
  label,
}: {
  children: ReactNode;
  field: string;
  label: string;
}) {
  return (
    <span className="inline-flex min-w-0 flex-[1_1_15rem] items-center gap-2">
      <label className={LABEL} htmlFor={field}>
        {label}
      </label>
      {children}
    </span>
  );
}

export function Choice({
  children,
  label,
  press,
  ready,
  strong,
  word,
}: {
  children?: ReactNode;
  label: string;
  press: () => void;
  ready?: boolean;
  strong?: boolean;
  word?: string;
}) {
  const button = (
    <Button
      aria-label={label}
      disabled={ready === false}
      onClick={press}
      size="compact"
      variant={strong ? "primary" : "secondary"}
    >
      {children}
      <span aria-hidden="true">{word ?? label}</span>
    </Button>
  );
  return word && word !== label ? (
    <Tooltip content={label}>{button}</Tooltip>
  ) : (
    button
  );
}

export function RowNote({
  children,
  id,
  refused,
}: {
  children: ReactNode;
  id?: string;
  refused?: boolean;
}) {
  return (
    <output
      className={cn(
        "basis-full px-1 font-prose text-meta empty:hidden",
        refused ? "text-stop" : "text-mute",
      )}
      id={id}
      role={refused ? "alert" : undefined}
    >
      {children}
    </output>
  );
}
