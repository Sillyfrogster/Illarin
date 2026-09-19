import type { QueuedSend, WorkConnectedApp } from "@/lib/api/query";
import { sendFailureLine } from "@/lib/work-send";

export type InstallStep = {
  id: "queued" | "collected" | "installed";
  label: string;
  standing: "done" | "now" | "later";
};

/** InstallTrack is what an extension page shows for one connected app the reader sent it to. */
export type InstallTrack = {
  app: WorkConnectedApp;
  steps: InstallStep[];
  note: string;
  stopped: string | null;
  live: boolean;
};

function steps(done: number, now: number | null, last: string): InstallStep[] {
  return (
    [
      { id: "queued", label: "Queued" },
      { id: "collected", label: "Picked up" },
      { id: "installed", label: last },
    ] as const
  ).map((step, index) => ({
    ...step,
    standing: index < done ? "done" : index === now ? "now" : "later",
  }));
}

type Standing = Omit<InstallTrack, "app">;

/** installTrack reads what the page can say about one connected app, or nothing when it has never been sent there. */
export function installTrack(app: WorkConnectedApp): InstallTrack | null {
  const standing = app.send
    ? sendStanding(app, app.send)
    : libraryStanding(app);
  return standing ? { app, ...standing } : null;
}

function libraryStanding(app: WorkConnectedApp): Standing | null {
  if (app.installedVersion === null) return null;
  const here = app.name;
  return {
    steps: [],
    note: app.updateAvailable
      ? `Installed on ${here}, and a newer version exists here.`
      : `Installed on ${here}.`,
    stopped: null,
    live: false,
  };
}

function sendStanding(app: WorkConnectedApp, send: QueuedSend): Standing {
  const here = app.name;
  const named = app.appName;
  const quiet = { stopped: null, live: false };
  switch (send.state) {
    case "queued":
      return {
        ...quiet,
        steps: steps(0, 0, "Installed"),
        note: `Waiting for ${here} to collect it.`,
        live: true,
      };
    case "released":
      return {
        ...quiet,
        steps: steps(1, 1, "Installed"),
        note: `${here} picked it up and is installing it.`,
        live: true,
      };
    case "delivered":
      return {
        ...quiet,
        steps: steps(3, null, send.updatesInstall ? "Updated" : "Installed"),
        note: send.updatesInstall
          ? `Updated on ${here}. It stays on, and ${named} asks only about permissions the update adds.`
          : `Installed on ${here}, switched off until you approve its permissions in ${named}.`,
      };
    default:
      return {
        ...quiet,
        steps: steps(0, null, "Installed"),
        note: "",
        stopped:
          send.reason === "unsupported"
            ? `${here} no longer says it installs extensions.`
            : sendFailureLine(send.reason),
      };
  }
}
