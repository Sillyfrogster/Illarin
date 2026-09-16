"use client";

import { Check, CircleAlert, SendHorizontal } from "lucide-react";
import { useState } from "react";
import { browserFetch } from "@/lib/api/browser-mutation";
import type { Notification } from "@/lib/api/notifications";
import { cn } from "@/lib/cn";

type SendTarget = NonNullable<Notification["sendTargets"]>[number];
type Progress = "ready" | "sending" | "waiting" | "failed";

const CHIP =
  "inline-flex min-h-11 max-w-full items-center gap-2 rounded-control px-3 text-meta font-medium outline-offset-2 transition-colors duration-200 motion-reduce:transition-none [&_svg]:size-3.5 [&_svg]:shrink-0";

const LOOK: Record<Progress, string> = {
  ready: "bg-deep text-ink hover:bg-accent-wash hover:text-accent",
  sending: "bg-deep text-mute",
  waiting: "bg-transparent text-mute inset-ring inset-ring-edge",
  failed: "bg-stop-wash text-stop hover:bg-stop-wash/80",
};

/** The sends an update entry offers, one for each of the reader's instances that holds an older copy. */
export function SendUpdates({
  assetId,
  targets,
}: {
  assetId: string;
  targets: SendTarget[];
}) {
  const [progress, setProgress] = useState<Record<string, Progress>>({});
  const stateOf = (target: SendTarget): Progress =>
    progress[target.instanceId] ?? (target.waiting ? "waiting" : "ready");
  const failed = targets.some((target) => stateOf(target) === "failed");

  async function send(target: SendTarget) {
    setProgress((current) => ({ ...current, [target.instanceId]: "sending" }));
    const landed = await queueDelivery(assetId, target.instanceId);
    setProgress((current) => ({
      ...current,
      [target.instanceId]: landed ? "waiting" : "failed",
    }));
  }

  return (
    <div className="pt-2 pr-3 pb-3 pl-[3.875rem]">
      <p className="text-label font-medium text-ink/70">Send this update to</p>
      <div className="mt-1.5 flex flex-wrap gap-1.5">
        {targets.map((target) => {
          const state = stateOf(target);
          return (
            <button
              aria-label={sendLabel(target, state)}
              className={cn(CHIP, LOOK[state])}
              disabled={state === "sending" || state === "waiting"}
              key={target.instanceId}
              onClick={() => void send(target)}
              title={`${target.applicationName} — ${target.instanceName}`}
              type="button"
            >
              <Mark state={state} />
              <span className="truncate">{target.instanceName}</span>
            </button>
          );
        })}
      </div>
      {failed ? (
        <output
          aria-live="polite"
          className="mt-1.5 block text-label text-stop"
        >
          That send did not go through. Try again.
        </output>
      ) : null}
    </div>
  );
}

function Mark({ state }: { state: Progress }) {
  if (state === "waiting") return <Check aria-hidden="true" />;
  if (state === "failed") return <CircleAlert aria-hidden="true" />;
  return (
    <SendHorizontal
      aria-hidden="true"
      className={state === "sending" ? "animate-pulse" : undefined}
    />
  );
}

function sendLabel(target: SendTarget, state: Progress): string {
  if (state === "waiting") {
    return `${target.instanceName} is waiting to collect this update`;
  }
  return `Send the update to ${target.instanceName}`;
}

async function queueDelivery(
  assetId: string,
  instanceId: string,
): Promise<boolean> {
  try {
    const response = await browserFetch(
      `/api/v1/assets/${encodeURIComponent(assetId)}/deliveries`,
      {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ instanceId }),
      },
    );
    return response.ok;
  } catch {
    return false;
  }
}
