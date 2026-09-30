"use client";

import { Check, Copy } from "lucide-react";
import { type ReactNode, useEffect, useState } from "react";
import { cn } from "@/lib/cn";
import { Button } from "./button";
import { Tooltip } from "./tooltip";

const CONFIRMATION_MS = 2000;

/** CopyButton copies text on a click and shows a check while it confirms; the icon form is named by a tooltip. */
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
  const button = (
    <Button
      className={className}
      size={children ? "default" : "icon"}
      variant={children ? "secondary" : "ghost"}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setFailed(false);
          setCopied(true);
        } catch {
          setFailed(true);
        }
      }}
    >
      <span className="sr-only">
        {failed ? `${label} could not be copied` : copied ? "Copied" : label}
      </span>
      <Icon aria-hidden="true" className={cn(copied && "text-accent")} />
      {children ? (
        <span aria-hidden="true">{copied ? "Copied" : children}</span>
      ) : null}
    </Button>
  );
  return children ? button : <Tooltip content={label}>{button}</Tooltip>;
}
