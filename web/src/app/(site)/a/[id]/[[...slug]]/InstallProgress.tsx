"use client";

import { Circle, CircleAlert, CircleCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Item, ItemGroup, ItemTitle } from "@/components/ui/item";
import { Spinner } from "@/components/ui/spinner";
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
        <ItemGroup as="ol" className="mt-3">
          {track.steps.map((step) => (
            <Step key={step.id} step={step} />
          ))}
        </ItemGroup>
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

function Step({ step }: { step: InstallStep }) {
  const done = step.standing === "done";
  const now = step.standing === "now";
  return (
    <Item className="flex-row items-center gap-3 py-2.5">
      {done ? (
        <CircleCheck
          aria-hidden="true"
          className="size-4 shrink-0 text-accent"
        />
      ) : now ? (
        <Spinner className="text-accent" />
      ) : (
        <Circle aria-hidden="true" className="size-4 shrink-0 text-edge" />
      )}
      <ItemTitle className={cn(!done && !now && "font-normal text-mute")}>
        {step.label}
        {now ? <span className="sr-only"> (now)</span> : null}
        {done ? <span className="sr-only"> (done)</span> : null}
      </ItemTitle>
    </Item>
  );
}
