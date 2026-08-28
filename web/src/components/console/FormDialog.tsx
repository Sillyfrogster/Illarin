"use client";

import { X } from "lucide-react";
import { type FormEvent, type ReactNode, useEffect, useRef } from "react";
import styles from "./FormDialog.module.css";

export function FormDialog({
  open,
  title,
  hint,
  commit,
  busy,
  ready = true,
  acknowledge,
  destructive,
  onClose,
  onCommit,
  children,
}: {
  open: boolean;
  title: string;
  hint?: string;
  commit: string;
  busy?: boolean;
  ready?: boolean;
  acknowledge?: boolean;
  destructive?: ReactNode;
  onClose: () => void;
  onCommit: () => void;
  children: ReactNode;
}) {
  const dialog = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const element = dialog.current;
    if (!element) return;
    if (open && !element.open) element.showModal();
    if (!open && element.open) element.close();
  }, [open]);

  function send(event: FormEvent) {
    event.preventDefault();
    if (!ready || busy) return;
    onCommit();
  }

  return (
    <dialog
      className={styles.dialog}
      ref={dialog}
      onClose={onClose}
      aria-label={title}
    >
      <form className={styles.body} onSubmit={send}>
        <header className={styles.heading}>
          <div>
            <h2>{title}</h2>
            {hint ? <p>{hint}</p> : null}
          </div>
          <button
            className={styles.dismiss}
            type="button"
            onClick={onClose}
            aria-label="Close"
          >
            <X size={17} strokeWidth={1.8} aria-hidden="true" />
          </button>
        </header>

        <div className={styles.fields}>{children}</div>

        <footer
          className={styles.commit}
          data-acknowledge={acknowledge || undefined}
        >
          {destructive ? (
            <div className={styles.destructive}>{destructive}</div>
          ) : null}
          {acknowledge ? null : (
            <button type="button" className={styles.cancel} onClick={onClose}>
              Cancel
            </button>
          )}
          <button
            type="submit"
            className={styles.keep}
            disabled={!ready || busy}
          >
            {commit}
          </button>
        </footer>
      </form>
    </dialog>
  );
}
