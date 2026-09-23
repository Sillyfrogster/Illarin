"use client";

import { Check, Copy } from "lucide-react";
import { type ReactNode, useEffect, useState } from "react";
import { cn } from "@/lib/cn";
import { buttonVariants } from "./button";

const CONFIRMATION_MS = 2000;

/** Copies text on a click and shows a check while it confirms */
export function CopyButton({
  children,
  text,
  label,
  className,
}: {
  children?: ReactNode;
  text: string;
  label: string;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timer = setTimeout(() => setCopied(false), CONFIRMATION_MS);
    return () => clearTimeout(timer);
  }, [copied]);

  const Icon = copied ? Check : Copy;
  return (
    <button
      className={cn(
        children
          ? buttonVariants({ variant: "outline" })
          : "flex size-11 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-deep hover:text-ink",
        className,
      )}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setFailed(false);
          setCopied(true);
        } catch {
          setFailed(true);
        }
      }}
      type="button"
    >
      <span className="sr-only">
        {failed ? `${label} could not be copied` : copied ? "Copied" : label}
      </span>
      <Icon
        aria-hidden="true"
        className={cn("size-4", copied && "text-accent")}
      />
      {children ? (
        <span aria-hidden="true">{copied ? "Copied" : children}</span>
      ) : null}
    </button>
  );
}
