"use client";

import { Check, Copy } from "lucide-react";
import { useState } from "react";

export function RevealOnce({
  carry,
  copied,
  onCopied,
  value,
}: {
  carry: string;
  copied: boolean;
  onCopied: (copied: boolean) => void;
  value: string;
}) {
  const [trouble, setTrouble] = useState("");

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      onCopied(true);
    } catch {
      setTrouble("Your browser would not let us copy. Select it and copy it.");
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded-plate bg-deep p-4">
      <code className="block rounded-control bg-plane px-3.5 py-3 font-mono text-meta break-all text-ink select-all">
        {value}
      </code>
      <button
        aria-live="polite"
        className="inline-flex min-h-11 items-center justify-center gap-2 self-start rounded-control bg-action px-5 font-ui text-ui font-medium text-on-accent outline-offset-3 hover:opacity-90"
        onClick={copy}
        type="button"
      >
        {copied ? (
          <Check aria-hidden="true" className="size-4" strokeWidth={2} />
        ) : (
          <Copy aria-hidden="true" className="size-4" strokeWidth={1.8} />
        )}
        {copied ? "Copied" : "Copy"}
      </button>
      <p
        aria-live="polite"
        className="max-w-[52ch] font-prose text-meta text-mute"
      >
        {trouble || carry}
      </p>
    </div>
  );
}
