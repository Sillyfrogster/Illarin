import type {
  AppFormat,
  DownloadFormat,
  QueuedSend,
  WorkBlock,
  WorkConnectedApp,
  WorkDetail,
  WorkImage,
} from "@/lib/api/query";

type RoleVerdict = DownloadFormat["roles"][number];

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

export type SendDestination = {
  id: string;
  label: string;
};

export const DOWNLOAD_DESTINATION = "file";

/** canSendWork says whether the API will send this work to a connected app. */
export function canSendWork(
  work: Pick<WorkDetail, "lifecycle" | "withhold">,
): boolean {
  return work.lifecycle === "published" && !work.withhold;
}

/** isWaiting says whether a send is still on its way to the connected app. */
export function isWaiting(send: QueuedSend | null | undefined): boolean {
  return send?.state === "queued" || send?.state === "released";
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
  unsupported: "This app accepts no format this work can be written in.",
  abandoned: "The app kept collecting this send without installing it.",
};

function costs(role: RoleVerdict): boolean {
  return role.verdict !== "carried";
}

const GALLERY_ROLE = "gallery";

function asLoss(
  role: RoleVerdict,
  app: AppNaming,
  apps: AppFormat[],
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
  apps: AppFormat[],
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
function whoShows(role: RoleVerdict, apps: AppFormat[]): string {
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

function nameList(apps: AppFormat[]): string {
  const names = apps.map((app) => app.label);
  if (names.length < 2) return names.join("");
  return `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
}

function costLine(
  offered: DownloadFormat,
  holdsNothing: boolean,
  app: AppNaming,
): string {
  const lost = app.id
    ? offered.roles.filter((role) => !reaches(role, app.id)).length
    : offered.roles.filter(costs).length;
  const elsewhere = app.id ? 0 : offered.roles.filter(landsElsewhere).length;
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
export function appLabel(apps: AppFormat[], id: string): string {
  return apps.find((app) => app.id === id)?.label ?? "";
}

export function formatChoices({
  downloads,
  holdsNothing,
  app = "",
  apps = [],
}: {
  downloads: DownloadFormat[];
  holdsNothing: boolean;
  app?: string;
  apps?: AppFormat[];
}): FormatChoice[] {
  const naming = { id: app, label: appLabel(apps, app) };
  const pick = app
    ? (apps.find((one) => one.id === app)?.format ?? "")
    : downloads.find((offered) => offered.recommended)?.format;
  const ordered = [
    ...downloads.filter((offered) => offered.format === pick),
    ...downloads.filter((offered) => offered.format !== pick),
  ];

  return ordered.map((offered) => {
    const noted = [
      ...offered.roles.filter(costs),
      ...offered.roles.filter((role) => !costs(role) && role.destination),
    ].map((role) => asLoss(role, naming, apps));
    const gallery = offered.roles.find((role) => role.role === GALLERY_ROLE);
    return {
      format: offered.format,
      label: offered.label,
      recommended: offered.format === pick,
      cost: costLine(offered, holdsNothing, naming),
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

export function sendDestinations(apps: WorkConnectedApp[]): SendDestination[] {
  return [
    { id: DOWNLOAD_DESTINATION, label: "Download a file" },
    ...apps
      .filter((app) => app.canReceive)
      .map((app) => ({
        id: app.connectedAppId,
        label: `${app.appName} — ${app.name}`,
      })),
  ];
}

/** installsInApp says whether an app installs this type rather than reading it as content. */
export function installsInApp(type: string): boolean {
  return type === "extension";
}

export function sendActionLabel(
  app: WorkConnectedApp,
  installs = false,
): string {
  if (isWaiting(app.send)) return "Waiting to be collected";
  if (installs) {
    if (app.updateAvailable) return `Update on ${app.name}`;
    if (app.installedVersion !== null) return `Install again on ${app.name}`;
    return `Install on ${app.name}`;
  }
  if (app.updateAvailable) return "Send the update";
  if (app.installedVersion !== null) return "Send again";
  return "Send";
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
