"use client";

import { Check, Copy } from "lucide-react";
import { useState } from "react";
import styles from "./RevealOnce.module.css";

/** A secret Illarin can only show once, with the one chance to take it. */
export function RevealOnce({
  value,
  carry,
  copied,
  onCopied,
}: {
  value: string;
  carry: string;
  copied: boolean;
  onCopied: (copied: boolean) => void;
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
    <div className={styles.reveal}>
      <code className={styles.value}>{value}</code>
      <button
        type="button"
        className={styles.copy}
        onClick={copy}
        aria-live="polite"
      >
        {copied ? (
          <Check size={15} strokeWidth={2} aria-hidden="true" />
        ) : (
          <Copy size={15} strokeWidth={1.8} aria-hidden="true" />
        )}
        {copied ? "Copied" : "Copy"}
      </button>
      <p className={styles.carry} aria-live="polite">
        {trouble || carry}
      </p>
    </div>
  );
}
