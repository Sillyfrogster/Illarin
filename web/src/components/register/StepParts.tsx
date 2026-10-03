"use client";

import type { LucideIcon } from "lucide-react";
import { type FormEvent, type ReactNode, useState } from "react";
import { Button } from "@/components/ui/button";
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
      <Button disabled={!ready || busy} type="submit" variant="primary">
        {busy ? "Saving…" : commit}
      </Button>
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
    <Button
      className={cn(tone === "stop" && "text-stop hover:text-stop")}
      disabled={busy}
      onClick={onClick}
    >
      {Icon ? <Icon aria-hidden="true" /> : null}
      {children}
    </Button>
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
        <Button
          aria-expanded={asking}
          className={cn(!asking && "text-stop hover:text-stop")}
          disabled={busy}
          onClick={() => (asking ? onConfirm() : setAsking(true))}
          variant={asking ? "stop" : "secondary"}
        >
          {asking ? confirm : action}
        </Button>
        {asking ? (
          <Button onClick={() => setAsking(false)} variant="ghost">
            Cancel
          </Button>
        ) : null}
      </div>
    </div>
  );
}
