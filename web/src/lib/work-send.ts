import type {
  AppFormat,
  DownloadFormat,
  QueuedSend,
  WorkConnectedApp,
  WorkDetail,
} from "@/lib/api/query";

/** canSendWork says whether the API will send this work to a connected app. */
export function canSendWork(
  work: Pick<WorkDetail, "lifecycle" | "takedown">,
): boolean {
  return work.lifecycle === "published" && !work.takedown;
}

/** isWaiting says whether a send is still on its way to the connected app. */
export function isWaiting(send: QueuedSend | null | undefined): boolean {
  return send?.state === "queued" || send?.state === "released";
}

const FAILURES: Record<string, string> = {
  withdrawn: "This work was withdrawn before it could be collected.",
  unsupported: "This app accepts no format this work can be written in.",
  abandoned: "The app kept collecting this send without installing it.",
};

/** readerAppFormat is the format written for the reader's own app, when the work offers one. */
export function readerAppFormat(
  apps: AppFormat[],
  readerApp: string | null,
): AppFormat | null {
  return apps.find((app) => app.id === readerApp) ?? null;
}

/** orderedFormats puts the reader's app's format first, else the recommended one. */
export function orderedFormats(
  downloads: DownloadFormat[],
  lead: string | null = null,
): DownloadFormat[] {
  const first =
    lead ?? downloads.find((offered) => offered.recommended)?.format ?? "";
  return [
    ...downloads.filter((offered) => offered.format === first),
    ...downloads.filter((offered) => offered.format !== first),
  ];
}

export type MainAction =
  | { kind: "send"; app: WorkConnectedApp }
  | { kind: "download"; format: DownloadFormat; label: string | null }
  | { kind: "original" };

/** mainAction picks what the work page's main button does: send to a connected app, else download for the chosen app, else the recommended format, else the owner's original. */
export function mainAction({
  connected,
  downloads,
  forApp,
  hasOriginal,
  readerApp,
}: {
  connected: WorkConnectedApp[];
  downloads: DownloadFormat[];
  forApp: AppFormat | null;
  hasOriginal: boolean;
  readerApp: string | null;
}): MainAction | null {
  const receiving = connected.filter((one) => one.canReceive);
  const chosen = receiving.find(
    (one) => one.appName.toLowerCase() === readerApp,
  );
  const app = chosen ?? (forApp ? null : receiving[0]);
  if (app) return { kind: "send", app };
  const format = forApp
    ? downloads.find((one) => one.format === forApp.format)
    : downloads[0];
  if (format) return { kind: "download", format, label: forApp?.label ?? null };
  return hasOriginal ? { kind: "original" } : null;
}

/** installsInApp says whether an app installs this type rather than reading it as content. */
export function installsInApp(type: string): boolean {
  return type === "extension";
}

export function sendActionLabel(
  app: WorkConnectedApp,
  installs = false,
): string {
  if (isWaiting(app.send)) return `Sent to ${app.name}`;
  if (installs) {
    if (app.updateAvailable) return `Update on ${app.name}`;
    if (app.installedVersion !== null) return `Install again on ${app.name}`;
    return `Install on ${app.name}`;
  }
  if (app.updateAvailable) return `Send the update to ${app.name}`;
  if (app.installedVersion !== null) return `Send again to ${app.name}`;
  return `Send to ${app.name}`;
}

export function connectedAppStanding(app: WorkConnectedApp): string {
  if (app.updateAvailable) {
    return "Installed, and a newer version exists here.";
  }
  if (app.installedVersion !== null) {
    return "Installed and up to date.";
  }
  if (!app.reportsLibrary) {
    return "This app does not report installed works. Installation status is unavailable.";
  }
  return "Not installed here yet.";
}

export function sendFailureLine(reason: string | null | undefined): string {
  return FAILURES[reason ?? ""] ?? "This send did not arrive.";
}

/** downloadAddress names the version where one is chosen. */
export function downloadAddress({
  workId,
  format,
  version,
}: {
  workId: string;
  format: string;
  version?: number;
}): string {
  const address = `/download/${workId}/${format}`;
  return version ? `${address}?version=${version}` : address;
}
