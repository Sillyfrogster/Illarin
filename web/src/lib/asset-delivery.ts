import type {
  AssetBlock,
  AssetImage,
  AssetInstance,
  DownloadTarget,
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
  withdrawn: "This asset was withdrawn before it could be collected.",
  unsupported:
    "This application accepts no format this asset can be written in.",
  abandoned: "The application kept taking this delivery without installing it.",
};

function costs(role: RoleVerdict): boolean {
  return role.verdict !== "carried";
}

const GALLERY_ROLE = "gallery";

function asLoss(role: RoleVerdict): FormatLoss {
  return {
    role: role.role,
    label: role.label,
    line: verdictLine(role),
    sample: role.sample,
  };
}

function landsElsewhere(role: RoleVerdict): boolean {
  return !costs(role) && Boolean(role.destination);
}

function verdictLine(role: RoleVerdict): string {
  if (role.verdict === "dropped") return "Not included.";
  if (role.verdict === "reduced") {
    return role.reason
      ? `Included, without ${role.reason}.`
      : "Included, but not in full.";
  }
  return role.destination ?? "";
}

function costLine(target: DownloadTarget, holdsNothing: boolean): string {
  const lost = target.roles.filter(costs).length;
  const elsewhere = target.roles.filter(landsElsewhere).length;
  if (holdsNothing && lost === 0) return "There is nothing in it yet";
  if (lost > 0) {
    return lost === 1 ? "1 thing left out" : `${lost} things left out`;
  }
  if (elsewhere > 0) {
    return elsewhere === 1
      ? "1 thing some apps will not show"
      : `${elsewhere} things some apps will not show`;
  }
  return "Includes everything";
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

  return ordered.map((target) => {
    const noted = [
      ...target.roles.filter(costs),
      ...target.roles.filter((role) => !costs(role) && role.destination),
    ].map(asLoss);
    const gallery = target.roles.find((role) => role.role === GALLERY_ROLE);
    return {
      format: target.format,
      label: target.label,
      recommended: target.recommended,
      cost: costLine(target, holdsNothing),
      carriesGallery: gallery?.verdict !== "dropped",
      gallery: gallery ? asLoss(gallery) : null,
      losses: noted.filter((loss) => loss.role !== GALLERY_ROLE),
    };
  });
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

function imageItems(blocks: AssetBlock[], role: string) {
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
  blocks: AssetBlock[];
  images: AssetImage[];
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

/** downloadBytes estimates the file one choice of images produces. */
export function downloadBytes({
  format,
  blocks,
  images,
  chosen,
  carries = true,
}: {
  format: string;
  blocks: AssetBlock[];
  images: AssetImage[];
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
