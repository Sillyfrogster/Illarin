"use client";

import { Check, Copy } from "lucide-react";
import { useEffect, useState } from "react";
import { cn } from "@/lib/cn";

const CONFIRMATION_MS = 2000;

export function CopyButton({
  text,
  label,
  className,
}: {
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

  return (
    <button
      className={cn(
        "flex size-11 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-deep hover:text-ink",
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
      {copied ? (
        <Check aria-hidden="true" className="size-4 text-accent" />
      ) : (
        <Copy aria-hidden="true" className="size-4" />
      )}
    </button>
  );
}
