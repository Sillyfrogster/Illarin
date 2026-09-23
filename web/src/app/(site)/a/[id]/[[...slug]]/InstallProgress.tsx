"use client";

import { CircleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";
import type { InstallStep, InstallTrack } from "@/lib/install-track";

/** InstallProgress shows where an extension is on its way to one of the reader's connected apps. */
export function InstallProgress({
  track,
  busy,
  onDismiss,
}: {
  track: InstallTrack;
  busy: boolean;
  onDismiss: () => void;
}) {
  const { app } = track;
  const name = `${app.appName} — ${app.name}`;

  return (
    <section aria-label={name} aria-live="polite" className="max-w-[42ch]">
      <p className="text-meta font-medium text-ink">{name}</p>
      {track.steps.length > 0 ? (
        <ol className="mt-3 flex list-none items-start">
          {track.steps.map((step, index) => (
            <Step
              key={step.id}
              last={index === track.steps.length - 1}
              step={step}
            />
          ))}
        </ol>
      ) : null}
      {track.stopped ? (
        <p className="mt-3 flex items-start gap-2 text-meta text-stop">
          <CircleAlert
            aria-hidden="true"
            className="mt-0.5 size-3.5 shrink-0"
          />
          {track.stopped}
        </p>
      ) : (
        <p className="mt-3 text-meta text-mute">{track.note}</p>
      )}
      {app.send && !track.live ? (
        <div className="mt-3">
          <Button
            className="-ml-3"
            disabled={busy}
            onClick={onDismiss}
            size="compact"
            variant="ghost"
          >
            Dismiss
          </Button>
        </div>
      ) : null}
    </section>
  );
}

function Step({ step, last }: { step: InstallStep; last: boolean }) {
  const done = step.standing === "done";
  const now = step.standing === "now";
  return (
    <li
      aria-current={now ? "step" : undefined}
      className={cn("min-w-0", last ? "flex-none" : "flex-1")}
    >
      <span className="flex items-center">
        <span
          aria-hidden="true"
          className={cn(
            "size-3 shrink-0 rounded-full border-2",
            done && "border-accent bg-accent",
            now &&
              "animate-pulse border-accent bg-accent-wash motion-reduce:animate-none",
            !done && !now && "border-edge bg-plane",
          )}
        />
        {last ? null : (
          <span
            aria-hidden="true"
            className={cn(
              "mx-1.5 h-0.5 flex-1 rounded-full",
              done ? "bg-accent" : "bg-rule",
            )}
          />
        )}
      </span>
      <span
        className={cn(
          "mt-1.5 block pr-2 text-meta",
          done || now ? "font-medium text-ink" : "text-mute",
        )}
      >
        {step.label}
        {now ? <span className="sr-only"> (now)</span> : null}
        {done ? <span className="sr-only"> (done)</span> : null}
      </span>
    </li>
  );
}
