import type { QueuedDelivery, WorkInstance } from "@/lib/api/query";
import { deliveryFailureLine } from "@/lib/work-delivery";

export type InstallStep = {
  id: "queued" | "collected" | "installed";
  label: string;
  standing: "done" | "now" | "later";
};

/** InstallTrack is what an extension page shows for one instance the reader sent it to. */
export type InstallTrack = {
  instance: WorkInstance;
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

type Standing = Omit<InstallTrack, "instance">;

/** installTrack reads what the page can say about one instance, or nothing when it has never been sent there. */
export function installTrack(instance: WorkInstance): InstallTrack | null {
  const standing = instance.delivery
    ? deliveryStanding(instance, instance.delivery)
    : libraryStanding(instance);
  return standing ? { instance, ...standing } : null;
}

function libraryStanding(instance: WorkInstance): Standing | null {
  if (instance.installedGeneration === null) return null;
  const here = instance.instanceName;
  return {
    steps: [],
    note: instance.updateAvailable
      ? `Installed on ${here}, and a newer version exists here.`
      : `Installed on ${here}.`,
    stopped: null,
    live: false,
  };
}

function deliveryStanding(
  instance: WorkInstance,
  delivery: QueuedDelivery,
): Standing {
  const here = instance.instanceName;
  const app = instance.applicationName;
  const quiet = { stopped: null, live: false };
  switch (delivery.state) {
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
        steps: steps(
          3,
          null,
          delivery.updatesInstall ? "Updated" : "Installed",
        ),
        note: delivery.updatesInstall
          ? `Updated on ${here}. It stays on, and ${app} asks only about permissions the update adds.`
          : `Installed on ${here}, switched off until you approve its permissions in ${app}.`,
      };
    default:
      return {
        ...quiet,
        steps: steps(0, null, "Installed"),
        note: "",
        stopped:
          delivery.reason === "unsupported"
            ? `${here} no longer says it installs extensions.`
            : deliveryFailureLine(delivery.reason),
      };
  }
}
