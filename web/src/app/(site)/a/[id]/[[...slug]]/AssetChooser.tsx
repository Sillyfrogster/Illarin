"use client";

import {
  CircleAlert,
  Clock,
  Download,
  FileDown,
  Images,
  Send,
  SlidersHorizontal,
} from "lucide-react";
import Image from "next/image";
import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { browserFetch } from "@/lib/api/browser-mutation";
import type {
  AppTarget,
  AssetBlock,
  AssetImage,
  AssetInstance,
  DownloadTarget,
  OriginalUpload,
  RecordedVersion,
} from "@/lib/api/query";
import {
  appLabel,
  DOWNLOAD_DESTINATION,
  deliveryDestinations,
  deliveryFailureLine,
  downloadAddress,
  downloadBytes,
  type FormatChoice,
  type FormatLoss,
  fileSize,
  formatChoices,
  installsOnInstance,
  instanceStanding,
  isWaiting,
  MAX_DOWNLOAD_BYTES,
  sendActionLabel,
  type TravellingImage,
  travellingGallery,
} from "@/lib/asset-delivery";
import { versionDate } from "@/lib/asset-updates";
import { cn } from "@/lib/cn";
import { shortMoment } from "@/lib/dates";
import { WatchOffer } from "./watch/WatchOffer";

function fileWord(mediaType: string): string {
  if (mediaType.startsWith("image/")) {
    return mediaType.slice("image/".length).toUpperCase();
  }
  if (mediaType === "application/json") return "JSON";
  if (mediaType === "application/zip") return "Archive";
  return "File";
}

function arrivalDate(when: string): string {
  return new Date(when).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}

export type AssetChooserProps = {
  assetId: string;
  kind: string;
  kindLabel: string;
  blocks: AssetBlock[];
  downloads: DownloadTarget[];
  appTargets: AppTarget[];
  original: OriginalUpload | null;
  images: AssetImage[];
  holdsNothing: boolean;
  isOwner: boolean;
  linkedInstallOnly: boolean;
};

/** AssetChooser is the body of the download chooser, from formats and images to the file itself. */
export function AssetChooser({
  assetId,
  kind,
  blocks,
  downloads,
  appTargets,
  original,
  images,
  holdsNothing,
  isOwner,
  linkedInstallOnly,
  instances,
  refresh,
  onSent,
  version = null,
}: AssetChooserProps & {
  instances: AssetInstance[];
  refresh: () => Promise<void>;
  onSent?: () => void;
  version?: RecordedVersion | null;
}) {
  const installs = installsOnInstance(kind);
  const [format, setFormat] = useState("");
  const [app, setApp] = useState(appTargets[0]?.id ?? "");
  const [openFormats, setOpenFormats] = useState(false);
  const [destination, setDestination] = useState(
    installs
      ? (instances.find((one) => one.canReceive)?.instanceId ??
          DOWNLOAD_DESTINATION)
      : DOWNLOAD_DESTINATION,
  );
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [offering, setOffering] = useState(false);
  const gallery = useMemo(
    () => travellingGallery({ blocks, images }),
    [blocks, images],
  );
  const [taken, setTaken] = useState<string[] | null>(null);

  const choices = formatChoices({
    downloads,
    holdsNothing,
    app,
    apps: appTargets,
  });
  const chosen =
    choices.find((choice) => choice.format === format) ?? choices[0];
  const goingToAnApp = Boolean(app);
  const destinations = deliveryDestinations(instances).filter(
    (one) => one.id !== DOWNLOAD_DESTINATION || !linkedInstallOnly,
  );
  const goingTo =
    destinations.find((one) => one.id === destination) ?? destinations[0];
  const instance = instances.find(
    (one) => one.instanceId === goingTo?.id && one.canReceive,
  );
  const pending = isWaiting(instance?.delivery);
  const carriesGallery = chosen?.carriesGallery ?? false;
  const included =
    taken ?? gallery.filter((one) => one.chosen).map((one) => one.mediaId);
  const bytes = downloadBytes({
    format: chosen?.format ?? "",
    blocks,
    images,
    chosen: included,
    carries: carriesGallery,
  });
  const oversized = bytes > MAX_DOWNLOAD_BYTES;

  async function act(path: string, method: string, body?: unknown) {
    setBusy(true);
    setFailure("");
    try {
      const response = await browserFetch(path, {
        method,
        credentials: "same-origin",
        headers: body ? { "Content-Type": "application/json" } : undefined,
        body: body ? JSON.stringify(body) : undefined,
      });
      if (!response.ok) {
        const answer: unknown = await response.json().catch(() => null);
        setFailure(
          typeof answer === "object" &&
            answer !== null &&
            "error" in answer &&
            typeof answer.error === "string"
            ? answer.error
            : "Illarin could not do that. Try again in a moment.",
        );
        return;
      }
      await refresh();
      return true;
    } catch {
      setFailure(
        "We could not reach Illarin. Check your connection and try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  const toAFile = goingTo?.id === DOWNLOAD_DESTINATION;

  return (
    <>
      {version ? <WrittenToday version={version} /> : null}

      {choices.length > 0 && !linkedInstallOnly ? (
        <>
          {appTargets.length === 0 ? (
            <fieldset className="min-w-0 border-0 p-0">
              <legend className="mb-2 text-meta font-medium text-ink">
                Format
              </legend>
              <div className="space-y-0.5">
                {choices.map((choice) => (
                  <FormatRow
                    chosen={choice.format === chosen?.format}
                    choice={choice}
                    key={choice.format}
                    onChoose={() => setFormat(choice.format)}
                  />
                ))}
              </div>
            </fieldset>
          ) : appTargets.length > 1 ? (
            <fieldset className="min-w-0 border-0 p-0">
              <legend className="mb-2 text-meta font-medium text-ink">
                Choose an app
              </legend>
              <div className="flex flex-wrap gap-1.5">
                {appTargets.map((one) => (
                  <AppChip
                    chosen={goingToAnApp && one.id === app}
                    key={one.id}
                    label={one.label}
                    onChoose={() => {
                      setApp(one.id);
                      setFormat("");
                    }}
                  />
                ))}
              </div>
            </fieldset>
          ) : null}

          {chosen ? (
            <WhatTravels
              choice={chosen}
              images={images}
              named={goingToAnApp ? appLabel(appTargets, app) : ""}
            />
          ) : null}
        </>
      ) : null}

      {gallery.length > 0 && !linkedInstallOnly ? (
        <GalleryChoice
          bytes={bytes}
          carried={carriesGallery}
          gallery={gallery}
          included={included}
          onChange={setTaken}
          oversized={oversized}
          verdict={chosen?.gallery ?? null}
        />
      ) : null}

      {destinations.length > 1 ? (
        <>
          <label
            className="mt-5 block text-meta font-medium text-ink"
            htmlFor="get-asset-destination"
          >
            {installs ? "Install on" : "Send to"}
          </label>
          <Select
            className="mt-2"
            disabled={busy}
            id="get-asset-destination"
            onChange={(event) => {
              setDestination(event.target.value);
              setFailure("");
            }}
            value={goingTo?.id}
          >
            {destinations.map((one) => (
              <option key={one.id} value={one.id}>
                {one.label}
              </option>
            ))}
          </Select>
        </>
      ) : null}

      {instance ? <InstanceStanding instance={instance} /> : null}

      {toAFile && oversized ? (
        <p className="mt-4 flex items-start gap-2 text-meta text-stop">
          <CircleAlert
            aria-hidden="true"
            className="mt-0.5 size-3.5 shrink-0"
          />
          Illarin writes files up to {fileSize(MAX_DOWNLOAD_BYTES)}. Select
          fewer gallery images to reduce the download size.
        </p>
      ) : null}

      {toAFile ? (
        <Button
          asChild={!oversized}
          className="mt-4 w-full"
          disabled={oversized}
          variant="primary"
        >
          {oversized ? (
            <>
              <Download aria-hidden="true" />
              {downloadLabel(
                chosen,
                goingToAnApp ? appLabel(appTargets, app) : "",
              )}
            </>
          ) : (
            <a
              onClick={() => setOffering(true)}
              href={downloadAddress({
                assetId,
                format: chosen?.format ?? "",
                carried: carriesGallery,
                gallery,
                included,
                version: version?.number,
              })}
            >
              <Download aria-hidden="true" />
              {downloadLabel(
                chosen,
                goingToAnApp ? appLabel(appTargets, app) : "",
              )}
            </a>
          )}
        </Button>
      ) : instance ? (
        <div className="mt-4 flex flex-wrap gap-2">
          <Button
            className="flex-1"
            disabled={pending}
            loading={busy}
            onClick={async () => {
              const sent = await act(
                `/api/v1/assets/${assetId}/deliveries`,
                "POST",
                { instanceId: instance.instanceId },
              );
              if (!sent) return;
              setOffering(true);
              onSent?.();
            }}
            variant="primary"
          >
            {pending ? (
              <Clock aria-hidden="true" />
            ) : (
              <Send aria-hidden="true" />
            )}
            {sendActionLabel(instance, installs)}
          </Button>
          {instance.delivery ? (
            <Button
              disabled={busy}
              onClick={() =>
                instance.delivery &&
                act(`/api/v1/deliveries/${instance.delivery.id}`, "DELETE")
              }
            >
              {pending ? "Cancel" : "Dismiss"}
            </Button>
          ) : null}
        </div>
      ) : null}

      {failure ? (
        <output aria-live="polite" className="mt-3 block text-meta text-stop">
          {failure}
        </output>
      ) : null}

      {offering ? <WatchOffer /> : null}

      {choices.length > 1 && !linkedInstallOnly && appTargets.length > 0 ? (
        <div className="mt-4 border-rule border-t pt-3">
          {openFormats ? (
            <fieldset className="min-w-0 border-0 p-0">
              <legend className="mb-2 text-meta font-medium text-ink">
                Format
              </legend>
              <div className="space-y-0.5">
                {choices.map((choice) => (
                  <FormatRow
                    chosen={choice.format === chosen?.format}
                    choice={choice}
                    key={choice.format}
                    onChoose={() => {
                      setFormat(choice.format);
                      setApp("");
                    }}
                  />
                ))}
              </div>
            </fieldset>
          ) : (
            <button
              className="inline-flex min-h-11 items-center gap-2 rounded-control px-2 text-meta font-medium text-ink outline-offset-3 hover:bg-deep"
              onClick={() => setOpenFormats(true)}
              type="button"
            >
              <SlidersHorizontal aria-hidden="true" size={15} />
              Choose a file format
            </button>
          )}
        </div>
      ) : null}

      {original && isOwner && !version ? (
        <div className="mt-5 border-rule border-t pt-4">
          <p className="text-meta font-medium text-ink">Original upload</p>
          <p className="mt-1 text-meta text-mute">
            {original.label ? `${original.label} · ` : ""}
            {fileWord(original.mediaType)}, uploaded{" "}
            {arrivalDate(original.arrivedAt)}. Edits made since are not in it.
          </p>
          <Button asChild className="mt-3 w-full">
            <a href={`/download/${assetId}`}>
              <FileDown aria-hidden="true" />
              Download the original
            </a>
          </Button>
        </div>
      ) : null}
    </>
  );
}

/** Explains how historical downloads are generated. */
function WrittenToday({ version }: { version: RecordedVersion }) {
  return (
    <p className="mb-4 max-w-[42ch] text-meta text-mute">
      Written now, by Illarin’s current exporter, from what this version
      recorded on {versionDate(version)}: its own wording, pictures and
      preserved data. It is not the file the creator uploaded then.
    </p>
  );
}

function downloadLabel(
  choice: FormatChoice | undefined,
  named: string,
): string {
  if (named) return `Download for ${named}`;
  return `Download ${choice?.label ?? ""}`;
}

function GalleryChoice({
  bytes,
  carried,
  gallery,
  included,
  onChange,
  oversized,
  verdict,
}: {
  bytes: number;
  carried: boolean;
  gallery: TravellingImage[];
  included: string[];
  onChange: (images: string[]) => void;
  oversized: boolean;
  verdict: FormatLoss | null;
}) {
  const [open, setOpen] = useState(false);

  return (
    <fieldset className="mt-5 min-w-0 border-0 p-0">
      <div className="flex items-baseline justify-between gap-3">
        <legend className="text-meta font-medium text-ink">Images</legend>
        <p
          className={cn(
            "shrink-0 text-meta tabular-nums",
            oversized ? "text-stop" : "text-mute",
          )}
        >
          {carried ? `${included.length} of ${gallery.length}` : gallery.length}
        </p>
      </div>
      {verdict?.line ? (
        <p className="mt-0.5 text-meta text-mute">{verdict.line}</p>
      ) : null}
      {carried ? (
        <div className="mt-2 flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
          <p className={cn("text-meta", oversized ? "text-stop" : "text-mute")}>
            About {fileSize(bytes)}
          </p>
          <button
            aria-expanded={open}
            className="inline-flex min-h-11 items-center gap-2 rounded-control px-2 text-meta font-medium text-ink outline-offset-3 hover:bg-deep"
            onClick={() => setOpen((shown) => !shown)}
            type="button"
          >
            <Images aria-hidden="true" size={15} />
            {open ? "Done choosing" : "Choose images"}
          </button>
        </div>
      ) : null}
      {carried && open ? (
        <>
          <ul className="mt-2 flex list-none flex-wrap gap-2">
            {gallery.map((picture) => {
              const taken = included.includes(picture.mediaId);
              return (
                <li key={picture.mediaId}>
                  <label
                    className={cn(
                      "flex min-h-11 cursor-pointer items-center gap-2 rounded-control px-2 py-1.5",
                      taken ? "bg-accent-wash" : "hover:bg-deep",
                    )}
                  >
                    <input
                      checked={taken}
                      className="size-4 shrink-0 accent-[var(--v-action)]"
                      onChange={() =>
                        onChange(
                          taken
                            ? included.filter((one) => one !== picture.mediaId)
                            : [...included, picture.mediaId],
                        )
                      }
                      type="checkbox"
                    />
                    {picture.thumbUrl ? (
                      <Image
                        alt=""
                        className="size-8 rounded-control object-cover"
                        height={32}
                        src={picture.thumbUrl}
                        unoptimized
                        width={32}
                      />
                    ) : null}
                    <span className="max-w-40 truncate text-meta text-ink">
                      {picture.name}
                    </span>
                  </label>
                </li>
              );
            })}
          </ul>
          <p className="mt-2 text-meta text-mute">
            This selection changes only your download. The cover and expressions
            are always included.
          </p>
        </>
      ) : null}
    </fieldset>
  );
}

function AppChip({
  chosen,
  label,
  onChoose,
}: {
  chosen: boolean;
  label: string;
  onChoose: () => void;
}) {
  return (
    <button
      aria-pressed={chosen}
      className={cn(
        "min-h-11 rounded-control px-3 text-ui outline-offset-3",
        chosen
          ? "bg-accent-wash font-medium text-ink"
          : "text-mute hover:bg-deep hover:text-ink",
      )}
      onClick={onChoose}
      type="button"
    >
      {label}
    </button>
  );
}

/** Lists the content included in the selected download. */
function WhatTravels({
  choice,
  images,
  named,
}: {
  choice: FormatChoice;
  images: AssetImage[];
  named: string;
}) {
  return (
    <div className="mt-3">
      {named ? (
        <p className="text-meta text-mute">
          {choice.cost} · {choice.label}
        </p>
      ) : null}
      {choice.losses.length > 0 ? (
        <ul className="mt-3 list-none space-y-3.5">
          {choice.losses.map((loss) => (
            <li key={loss.role}>
              <div className="flex items-baseline justify-between gap-3">
                <p className="text-meta font-medium text-ink">{loss.label}</p>
                <p className="shrink-0 text-meta text-mute tabular-nums">
                  {loss.sample.count}
                </p>
              </div>
              <p className="mt-0.5 text-meta text-mute">{loss.line}</p>
              <Sample images={images} sample={loss.sample} />
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function FormatRow({
  choice,
  chosen,
  onChoose,
}: {
  choice: FormatChoice;
  chosen: boolean;
  onChoose: () => void;
}) {
  return (
    <div className={cn("rounded-control", chosen && "bg-accent-wash")}>
      <label className="flex min-h-12 cursor-pointer items-center gap-3 px-3 py-2">
        <input
          checked={chosen}
          className="size-4 shrink-0 accent-[var(--v-action)]"
          name="asset-format"
          onChange={onChoose}
          type="radio"
        />
        <span className="min-w-0 text-ui text-ink">
          {choice.label}
          <span className="block text-meta text-mute">{choice.cost}</span>
        </span>
      </label>
    </div>
  );
}

function Sample({
  sample,
  images,
}: {
  sample: FormatChoice["losses"][number]["sample"];
  images: AssetImage[];
}) {
  const imagesById = new Map(images.map((image) => [image.id, image]));
  const pictures = (sample.images ?? [])
    .map((id) => imagesById.get(id))
    .filter((image): image is AssetImage => image !== undefined);

  if (pictures.length > 0) {
    return (
      <ul className="mt-2 flex list-none flex-wrap gap-1.5">
        {pictures.map((picture) => (
          <li key={picture.id}>
            <Image
              alt=""
              className="size-10 rounded-control object-cover"
              height={40}
              src={picture.thumbUrl}
              unoptimized
              width={40}
            />
          </li>
        ))}
      </ul>
    );
  }

  const texts = sample.texts ?? [];
  if (texts.length === 0) return null;
  return (
    <ul className="mt-2 list-none space-y-1">
      {texts.map((text, index) => (
        <li
          className="truncate text-meta text-mute italic"
          key={`${index}-${text}`}
        >
          {text}
        </li>
      ))}
    </ul>
  );
}

function InstanceStanding({ instance }: { instance: AssetInstance }) {
  const delivery = instance.delivery;

  if (isWaiting(delivery)) {
    return (
      <p className="mt-4 flex items-start gap-2 text-meta text-mute">
        <Clock aria-hidden="true" className="mt-0.5 size-3.5 shrink-0" />
        Waiting for {instance.instanceName} to collect it.
      </p>
    );
  }
  if (delivery?.state === "failed") {
    return (
      <p className="mt-4 flex items-start gap-2 text-meta text-stop">
        <CircleAlert aria-hidden="true" className="mt-0.5 size-3.5 shrink-0" />
        {deliveryFailureLine(delivery.reason)}
      </p>
    );
  }
  return (
    <p className="mt-4 text-meta text-mute">
      {delivery?.state === "delivered" && delivery.settledAt
        ? `Delivered ${shortMoment(delivery.settledAt)}. `
        : ""}
      {instanceStanding(instance)}
    </p>
  );
}
