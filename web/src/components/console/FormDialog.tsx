"use client";

import { X } from "lucide-react";
import { type FormEvent, type ReactNode, useEffect, useRef } from "react";

/** A short form that interrupts the page, for a change that has nowhere else to go. */
export function FormDialog({
  busy,
  children,
  commit,
  destructive,
  hint,
  onClose,
  onCommit,
  open,
  ready = true,
  title,
}: {
  busy?: boolean;
  children: ReactNode;
  commit: string;
  /** A way to undo the thing entirely, kept apart from the commit. */
  destructive?: ReactNode;
  hint?: string;
  onClose: () => void;
  onCommit: () => void;
  open: boolean;
  ready?: boolean;
  title: string;
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
      aria-label={title}
      className="m-auto w-[min(34rem,calc(100vw-2rem))] rounded-plate border-0 bg-plane p-0 text-ink shadow-popover backdrop:bg-black/45"
      onClose={onClose}
      ref={dialog}
    >
      <form className="flex flex-col gap-6 p-6 md:p-7" onSubmit={send}>
        <header className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <h2 className="font-display text-section font-medium tracking-tight text-ink wrap-anywhere">
              {title}
            </h2>
            {hint ? (
              <p className="mt-2 font-prose text-meta text-mute">{hint}</p>
            ) : null}
          </div>
          <button
            aria-label="Close"
            className="inline-flex size-11 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-deep hover:text-ink"
            onClick={onClose}
            type="button"
          >
            <X aria-hidden="true" className="size-4.5" strokeWidth={1.8} />
          </button>
        </header>

        <div className="flex max-h-[60dvh] flex-col gap-5 overflow-y-auto">
          {children}
        </div>

        <footer className="flex flex-wrap items-center gap-2">
          {destructive ? <div className="mr-auto">{destructive}</div> : null}
          <button
            className="inline-flex min-h-11 items-center justify-center rounded-control px-4 font-ui text-ui font-medium text-mute outline-offset-3 hover:text-ink"
            onClick={onClose}
            type="button"
          >
            Cancel
          </button>
          <button
            className="inline-flex min-h-11 items-center justify-center rounded-control bg-action px-5 font-ui text-ui font-medium text-on-accent outline-offset-3 hover:opacity-90 disabled:opacity-40"
            disabled={!ready || busy}
            type="submit"
          >
            {busy ? "Saving…" : commit}
          </button>
        </footer>
      </form>
    </dialog>
  );
}
