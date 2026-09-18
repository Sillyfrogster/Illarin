import type {
  AppTarget,
  DownloadTarget,
  QueuedDelivery,
  WorkBlock,
  WorkDetail,
  WorkImage,
  WorkInstance,
} from "@/lib/api/query";

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
  carriesGallery: boolean;
  gallery: FormatLoss | null;
  losses: FormatLoss[];
};

export type DeliveryDestination = {
  id: string;
  label: string;
};

export const DOWNLOAD_DESTINATION = "file";

/** canSendWork says whether the API will send this work to an installation. */
export function canSendWork(
  work: Pick<WorkDetail, "lifecycle" | "withhold">,
): boolean {
  return work.lifecycle === "published" && !work.withhold;
}

/** isWaiting says whether a delivery is still on its way to the instance. */
export function isWaiting(
  delivery: QueuedDelivery | null | undefined,
): boolean {
  return delivery?.state === "queued" || delivery?.state === "released";
}

/** The largest file Illarin will produce, matching the API's own ceiling. */
export const MAX_DOWNLOAD_BYTES = 64 * 1024 * 1024;

/** What base64 costs a card that writes its images into the JSON itself. */
const BASE64_WEIGHT = 4 / 3;

const ARCHIVE_FORMATS = ["charx"];

export type TravellingImage = {
  mediaId: string;
  name: string;
  chosen: boolean;
  bytes: number;
  thumbUrl: string;
};

const FAILURES: Record<string, string> = {
  withdrawn: "This work was withdrawn before it could be collected.",
  unsupported:
    "This application accepts no format this work can be written in.",
  abandoned: "The application kept taking this delivery without installing it.",
};

function costs(role: RoleVerdict): boolean {
  return role.verdict !== "carried";
}

const GALLERY_ROLE = "gallery";

function asLoss(
  role: RoleVerdict,
  app: AppNaming,
  apps: AppTarget[],
): FormatLoss {
  return {
    role: role.role,
    label: role.label,
    line: verdictLine(role, app, apps),
    sample: role.sample,
  };
}

function landsElsewhere(role: RoleVerdict): boolean {
  return !costs(role) && Boolean(role.destination);
}

/** reaches says whether what a role wrote arrives in the named app. */
function reaches(role: RoleVerdict, app: string): boolean {
  if (costs(role)) return false;
  if (!role.destination) return true;
  return (role.shownBy ?? []).includes(app);
}

function verdictLine(
  role: RoleVerdict,
  app: AppNaming,
  apps: AppTarget[],
): string {
  if (role.verdict === "dropped") return "Not included.";
  if (role.verdict === "reduced") {
    return role.reason
      ? `Included, without ${role.reason}.`
      : "Included, but not in full.";
  }
  if (!role.destination) return "";
  if (app.id && app.label) {
    return reaches(role, app.id) ? "" : `${app.label} does not show these.`;
  }
  const named = whoShows(role, apps);
  return named ? `${role.destination} ${named}` : role.destination;
}

/** whoShows names the listed apps a destination reaches and the ones it does not. */
function whoShows(role: RoleVerdict, apps: AppTarget[]): string {
  const shows = apps.filter((app) => (role.shownBy ?? []).includes(app.id));
  const blind = apps.filter((app) => !(role.shownBy ?? []).includes(app.id));
  if (blind.length === 0) return "";
  const does = blind.length === 1 ? "does" : "do";
  if (shows.length === 0) {
    return `${nameList(blind)} ${does} not show these.`;
  }
  const verb = shows.length === 1 ? "shows" : "show";
  return `${nameList(shows)} ${verb} these; ${nameList(blind)} ${does} not.`;
}

function nameList(apps: AppTarget[]): string {
  const names = apps.map((app) => app.label);
  if (names.length < 2) return names.join("");
  return `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
}

function costLine(
  target: DownloadTarget,
  holdsNothing: boolean,
  app: AppNaming,
): string {
  const lost = app.id
    ? target.roles.filter((role) => !reaches(role, app.id)).length
    : target.roles.filter(costs).length;
  const elsewhere = app.id ? 0 : target.roles.filter(landsElsewhere).length;
  if (holdsNothing && lost === 0) return "No exportable content yet";
  if (lost > 0) {
    return lost === 1
      ? "1 content type has limited support"
      : `${lost} content types have limited support`;
  }
  if (elsewhere > 0) {
    return elsewhere === 1
      ? "1 content type unsupported by some apps"
      : `${elsewhere} content types unsupported by some apps`;
  }
  return "Includes everything";
}

type AppNaming = { id: string; label: string };

/** appLabel names an app the way the API labelled it. */
export function appLabel(apps: AppTarget[], id: string): string {
  return apps.find((app) => app.id === id)?.label ?? "";
}

export function formatChoices({
  downloads,
  holdsNothing,
  app = "",
  apps = [],
}: {
  downloads: DownloadTarget[];
  holdsNothing: boolean;
  app?: string;
  apps?: AppTarget[];
}): FormatChoice[] {
  const naming = { id: app, label: appLabel(apps, app) };
  const pick = app
    ? (apps.find((one) => one.id === app)?.format ?? "")
    : downloads.find((target) => target.recommended)?.format;
  const ordered = [
    ...downloads.filter((target) => target.format === pick),
    ...downloads.filter((target) => target.format !== pick),
  ];

  return ordered.map((target) => {
    const noted = [
      ...target.roles.filter(costs),
      ...target.roles.filter((role) => !costs(role) && role.destination),
    ].map((role) => asLoss(role, naming, apps));
    const gallery = target.roles.find((role) => role.role === GALLERY_ROLE);
    return {
      format: target.format,
      label: target.label,
      recommended: target.format === pick,
      cost: costLine(target, holdsNothing, naming),
      carriesGallery: gallery
        ? naming.id
          ? reaches(gallery, naming.id)
          : gallery.verdict !== "dropped"
        : false,
      gallery: gallery ? asLoss(gallery, naming, apps) : null,
      losses: noted.filter((loss) => loss.role !== GALLERY_ROLE),
    };
  });
}

export function deliveryDestinations(
  instances: WorkInstance[],
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

/** installsOnInstance says whether an app installs this type rather than reading it as content. */
export function installsOnInstance(type: string): boolean {
  return type === "extension";
}

export function sendActionLabel(
  instance: WorkInstance,
  installs = false,
): string {
  if (isWaiting(instance.delivery)) return "Waiting to be collected";
  if (installs) {
    if (instance.updateAvailable) return `Update on ${instance.instanceName}`;
    if (instance.installedGeneration !== null) {
      return `Install again on ${instance.instanceName}`;
    }
    return `Install on ${instance.instanceName}`;
  }
  if (instance.updateAvailable) return "Send the update";
  if (instance.installedGeneration !== null) return "Send again";
  return "Send";
}

export function instanceStanding(instance: WorkInstance): string {
  if (instance.updateAvailable) {
    return "Installed, and a newer version exists here.";
  }
  if (instance.installedGeneration !== null) {
    return "Installed and up to date.";
  }
  if (!instance.reportsLibrary) {
    return "This application does not report installed works. Installation status is unavailable.";
  }
  return "Not installed here yet.";
}

export function deliveryFailureLine(reason: string | null | undefined): string {
  return FAILURES[reason ?? ""] ?? "This delivery did not arrive.";
}

function imageItems(blocks: WorkBlock[], role: string) {
  return blocks
    .flatMap((block) => block.elements)
    .filter((element) => element.role === role)
    .flatMap((element) =>
      "images" in element.content ? element.content.images : [],
    );
}

/** travellingGallery lists the gallery in page order beside the creator's own choice. */
export function travellingGallery({
  blocks,
  images,
}: {
  blocks: WorkBlock[];
  images: WorkImage[];
}): TravellingImage[] {
  const stored = new Map(images.map((image) => [image.id, image]));
  return imageItems(blocks, "gallery").map((item, index) => ({
    mediaId: item.mediaId,
    name: item.name?.trim() || `Image ${index + 1}`,
    chosen: item.omitFromDownloads !== true,
    bytes: stored.get(item.mediaId)?.bytes ?? 0,
    thumbUrl: stored.get(item.mediaId)?.thumbUrl ?? "",
  }));
}

/** downloadAddress names the version where one is chosen, and the reader's images only where they differ from the creator's. */
export function downloadAddress({
  workId,
  format,
  carried,
  gallery,
  included,
  version,
}: {
  workId: string;
  format: string;
  carried: boolean;
  gallery: TravellingImage[];
  included: string[];
  version?: number;
}): string {
  const query = new URLSearchParams();
  if (version) query.set("version", String(version));
  const byDefault = gallery
    .filter((one) => one.chosen)
    .map((one) => one.mediaId);
  const asChosen =
    byDefault.length === included.length &&
    byDefault.every((mediaId) => included.includes(mediaId));
  if (carried && !asChosen) query.set("images", included.join(","));
  const address = `/download/${workId}/${format}`;
  const written = query.toString();
  return written ? `${address}?${written}` : address;
}

/** downloadBytes estimates the file one choice of images produces. */
export function downloadBytes({
  format,
  blocks,
  images,
  chosen,
  carries = true,
}: {
  format: string;
  blocks: WorkBlock[];
  images: WorkImage[];
  chosen: string[];
  carries?: boolean;
}): number {
  const stored = new Map(images.map((image) => [image.id, image]));
  const travelling = images
    .filter((image) => image.isCover)
    .map((one) => one.id);
  for (const item of imageItems(blocks, "expressions")) {
    travelling.push(item.mediaId);
  }
  if (carries) travelling.push(...chosen);
  const total = travelling.reduce(
    (sum, mediaId) => sum + (stored.get(mediaId)?.bytes ?? 0),
    0,
  );
  if (ARCHIVE_FORMATS.includes(format)) return total;
  return Math.round(total * BASE64_WEIGHT);
}

/** fileSize writes a byte count the way a person would say it. */
export function fileSize(bytes: number): string {
  const kilobytes = bytes / 1024;
  if (kilobytes < 1024) return `${Math.round(kilobytes)} KB`;
  const megabytes = kilobytes / 1024;
  return `${megabytes < 10 ? megabytes.toFixed(1) : Math.round(megabytes)} MB`;
}
