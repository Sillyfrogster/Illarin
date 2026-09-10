import type { AssetInstance, DownloadTarget } from "@/lib/api/query";

type RoleVerdict = DownloadTarget["roles"][number];

export type FormatLoss = {
  role: string;
  label: string;
  line: string;
  sample: RoleVerdict["sample"];
};

export type FormatChoice = {
  format: string;
  label: string;
  recommended: boolean;
  cost: string;
  losses: FormatLoss[];
};

export type DeliveryDestination = {
  id: string;
  label: string;
};

export const DOWNLOAD_DESTINATION = "file";

const FAILURES: Record<string, string> = {
  withdrawn: "This asset was withdrawn before it could be collected.",
  unsupported:
    "This application accepts no format this asset can be written in.",
  abandoned: "The application kept taking this delivery without installing it.",
};

function costs(role: RoleVerdict): boolean {
  return role.verdict !== "carried";
}

function verdictLine(role: RoleVerdict): string {
  if (role.verdict === "dropped") return "Not included.";
  if (role.verdict === "reduced") {
    return role.reason
      ? `Included, without ${role.reason}.`
      : "Included, but not in full.";
  }
  return role.destination ? `Included as ${role.destination}.` : "";
}

function costLine(target: DownloadTarget, holdsNothing: boolean): string {
  const lost = target.roles.filter(costs).length;
  if (holdsNothing && lost === 0) return "There is nothing in it yet";
  if (lost === 0) return "Includes everything";
  return lost === 1 ? "1 thing left out" : `${lost} things left out`;
}

export function formatChoices({
  downloads,
  holdsNothing,
}: {
  downloads: DownloadTarget[];
  holdsNothing: boolean;
}): FormatChoice[] {
  const ordered = [
    ...downloads.filter((target) => target.recommended),
    ...downloads.filter((target) => !target.recommended),
  ];

  return ordered.map((target) => ({
    format: target.format,
    label: target.label,
    recommended: target.recommended,
    cost: costLine(target, holdsNothing),
    losses: [
      ...target.roles.filter(costs),
      ...target.roles.filter((role) => !costs(role) && role.destination),
    ].map((role) => ({
      role: role.role,
      label: role.label,
      line: verdictLine(role),
      sample: role.sample,
    })),
  }));
}

export function deliveryDestinations(
  instances: AssetInstance[],
): DeliveryDestination[] {
  return [
    { id: DOWNLOAD_DESTINATION, label: "Download a file" },
    ...instances
      .filter((instance) => instance.canReceive)
      .map((instance) => ({
        id: instance.instanceId,
        label: `${instance.applicationName} — ${instance.instanceName}`,
      })),
  ];
}

export function sendActionLabel(instance: AssetInstance): string {
  if (instance.delivery && instance.delivery.state !== "failed") {
    return "Waiting to be collected";
  }
  if (instance.updateAvailable) return "Send the update";
  if (instance.installedGeneration !== null) return "Send again";
  return "Send";
}

export function instanceStanding(instance: AssetInstance): string {
  if (instance.updateAvailable) {
    return "Installed, and a newer version exists here.";
  }
  if (instance.installedGeneration !== null) {
    return "Installed and up to date.";
  }
  if (!instance.reportsLibrary) {
    return "This installation does not report what it holds, so Illarin cannot say whether you already have it.";
  }
  return "Not installed here yet.";
}

export function deliveryFailureLine(reason: string | null | undefined): string {
  return FAILURES[reason ?? ""] ?? "This delivery did not arrive.";
}

export function itemCount(count: number): string {
  return count === 1 ? "1 item" : `${count} items`;
}
