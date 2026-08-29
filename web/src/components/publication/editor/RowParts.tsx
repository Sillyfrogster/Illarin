"use client";

import type { ReactNode } from "react";
import styles from "./Toolbar.module.css";

/** The strip under the formatting bar. One is open at a time. */
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
      className={styles.row}
      onKeyDown={(event) => {
        if (event.key === "Escape" && onClose) onClose();
      }}
    >
      <span aria-hidden="true" className={styles.rowMark}>
        {label}
      </span>
      {children}
    </fieldset>
  );
}

/** A label and its field, kept together when the row wraps. */
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
    <span className={styles.pair}>
      <label className={styles.pickerLabel} htmlFor={field}>
        {label}
      </label>
      {children}
    </span>
  );
}

/** One thing a row does. */
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
  return (
    <button
      aria-label={label}
      className={strong ? `${styles.choice} ${styles.strong}` : styles.choice}
      disabled={ready === false}
      onClick={press}
      title={label}
      type="button"
    >
      {children}
      <span aria-hidden="true">{word ?? label}</span>
    </button>
  );
}
