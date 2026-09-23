"use client";

import { Check, CircleAlert, SendHorizontal } from "lucide-react";
import { useState } from "react";
import { api } from "@/lib/api/client";
import type { Notification } from "@/lib/api/notifications";
import { cn } from "@/lib/cn";

type SendApp = NonNullable<Notification["sendTo"]>[number];
type Progress = "ready" | "sending" | "waiting" | "failed";

const CHIP =
  "inline-flex min-h-11 max-w-full items-center gap-2 rounded-control px-3 text-meta font-medium outline-offset-2 transition-colors duration-200 motion-reduce:transition-none [&_svg]:size-3.5 [&_svg]:shrink-0";

const LOOK: Record<Progress, string> = {
  ready: "bg-deep text-ink hover:bg-accent-wash hover:text-accent",
  sending: "bg-deep text-mute",
  waiting: "bg-transparent text-mute inset-ring inset-ring-edge",
  failed: "bg-stop-wash text-stop hover:bg-stop-wash/80",
};

/** The sends an update entry offers, one for each of the reader's connected apps that holds an older copy. */
export function SendUpdates({
  workId,
  apps,
}: {
  workId: string;
  apps: SendApp[];
}) {
  const [progress, setProgress] = useState<Record<string, Progress>>({});
  const stateOf = (app: SendApp): Progress =>
    progress[app.connectedAppId] ?? (app.waiting ? "waiting" : "ready");
  const failed = apps.some((app) => stateOf(app) === "failed");

  async function send(app: SendApp) {
    setProgress((current) => ({ ...current, [app.connectedAppId]: "sending" }));
    const landed = await queueSend(workId, app.connectedAppId);
    setProgress((current) => ({
      ...current,
      [app.connectedAppId]: landed ? "waiting" : "failed",
    }));
  }

  return (
    <div className="pt-2 pr-3 pb-3 pl-[3.875rem]">
      <p className="text-label font-medium text-ink/70">Send this update to</p>
      <div className="mt-1.5 flex flex-wrap gap-1.5">
        {apps.map((app) => {
          const state = stateOf(app);
          return (
            <button
              aria-label={sendLabel(app, state)}
              className={cn(CHIP, LOOK[state])}
              disabled={state === "sending" || state === "waiting"}
              key={app.connectedAppId}
              onClick={() => void send(app)}
              title={`${app.appName} — ${app.name}`}
              type="button"
            >
              <Mark state={state} />
              <span className="truncate">{app.name}</span>
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

function sendLabel(app: SendApp, state: Progress): string {
  if (state === "waiting") {
    return `${app.name} is waiting to collect this update`;
  }
  return `Send the update to ${app.name}`;
}

async function queueSend(
  workId: string,
  connectedAppId: string,
): Promise<boolean> {
  try {
    const { response } = await api<unknown>(
      "POST",
      `/v1/works/${encodeURIComponent(workId)}/sends`,
      { body: { connectedAppId } },
    );
    return response.ok;
  } catch {
    return false;
  }
}
