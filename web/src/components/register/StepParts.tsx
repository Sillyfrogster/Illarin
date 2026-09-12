"use client";

import type { LucideIcon } from "lucide-react";
import { type FormEvent, type ReactNode, useState } from "react";
import { cn } from "@/lib/cn";

export function StepForm({
  busy,
  children,
  commit,
  onCommit,
  ready = true,
  under,
}: {
  busy?: boolean;
  children: ReactNode;
  commit: string;
  onCommit: () => void;
  ready?: boolean;
  under?: ReactNode;
}) {
  function send(event: FormEvent) {
    event.preventDefault();
    if (!ready || busy) return;
    onCommit();
  }

  return (
    <form className="flex flex-col gap-6" onSubmit={send}>
      <div className="flex flex-col gap-5">{children}</div>
      <button
        className="inline-flex min-h-11 items-center justify-center rounded-control bg-action px-5 font-ui text-ui font-medium text-on-accent outline-offset-3 hover:opacity-90 disabled:opacity-40"
        disabled={!ready || busy}
        type="submit"
      >
        {busy ? "Saving…" : commit}
      </button>
      {under}
    </form>
  );
}

export function StepNote({
  children,
  tone = "quiet",
}: {
  children: ReactNode;
  tone?: "quiet" | "stop";
}) {
  return (
    <p
      className={cn(
        "max-w-[52ch] font-prose text-meta",
        tone === "stop" ? "text-stop" : "text-mute",
      )}
    >
      {children}
    </p>
  );
}

export function StepAction({
  busy,
  children,
  icon: Icon,
  onClick,
  tone = "quiet",
}: {
  busy?: boolean;
  children: ReactNode;
  icon?: LucideIcon;
  onClick: () => void;
  tone?: "quiet" | "stop";
}) {
  return (
    <button
      className={cn(
        "inline-flex min-h-11 items-center justify-center gap-2 rounded-control px-4 font-ui text-ui font-medium outline-offset-3 disabled:opacity-40",
        tone === "stop"
          ? "bg-stop-wash text-stop hover:opacity-85"
          : "bg-deep text-ink hover:bg-rule/45",
      )}
      disabled={busy}
      onClick={onClick}
      type="button"
    >
      {Icon ? (
        <Icon aria-hidden="true" className="size-4" strokeWidth={1.9} />
      ) : null}
      {children}
    </button>
  );
}

export function Consequence({
  action,
  busy,
  children,
  confirm,
  onConfirm,
}: {
  action: string;
  busy?: boolean;
  children: ReactNode;
  confirm: string;
  onConfirm: () => void;
}) {
  const [asking, setAsking] = useState(false);

  return (
    <div className="flex flex-col gap-3 border-t border-rule/45 pt-5">
      {asking ? (
        <p className="max-w-[52ch] font-prose text-meta text-stop" role="alert">
          {children}
        </p>
      ) : null}
      <div className="flex flex-wrap gap-2">
        <button
          aria-expanded={asking}
          className="inline-flex min-h-11 items-center justify-center rounded-control bg-stop-wash px-4 font-ui text-ui font-medium text-stop outline-offset-3 hover:opacity-85 disabled:opacity-40"
          disabled={busy}
          onClick={() => (asking ? onConfirm() : setAsking(true))}
          type="button"
        >
          {asking ? confirm : action}
        </button>
        {asking ? (
          <button
            className="inline-flex min-h-11 items-center justify-center rounded-control px-4 font-ui text-ui font-medium text-mute outline-offset-3 hover:text-ink"
            onClick={() => setAsking(false)}
            type="button"
          >
            Cancel
          </button>
        ) : null}
      </div>
    </div>
  );
}
