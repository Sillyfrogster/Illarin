"use client";

import {
  ChevronDown,
  CircleAlert,
  Clock,
  Download,
  FileDown,
  Images,
  Send,
  SlidersHorizontal,
} from "lucide-react";
import Image from "next/image";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Select } from "@/components/ui/select";
import { browserFetch } from "@/lib/api/browser-mutation";
import type {
  AppTarget,
  AssetBlock,
  AssetImage,
  AssetInstance,
  AssetInstanceList,
  DownloadTarget,
  OriginalUpload,
} from "@/lib/api/query";
import {
  appLabel,
  DOWNLOAD_DESTINATION,
  deliveryDestinations,
  deliveryFailureLine,
  downloadBytes,
  type FormatChoice,
  type FormatLoss,
  fileSize,
  formatChoices,
  instanceStanding,
  MAX_DOWNLOAD_BYTES,
  sendActionLabel,
  type TravellingImage,
  travellingGallery,
} from "@/lib/asset-delivery";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";

const WATCH_INTERVAL_MS = 8000;
const WATCH_LIMIT = 20;

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

export function GetAsset({
  assetId,
  kindLabel,
  blocks,
  downloads,
  appTargets,
  original,
  images,
  holdsNothing,
  isOwner,
  linkedInstallOnly,
}: {
  assetId: string;
  kindLabel: string;
  blocks: AssetBlock[];
  downloads: DownloadTarget[];
  appTargets: AppTarget[];
  original: OriginalUpload | null;
  images: AssetImage[];
  holdsNothing: boolean;
  isOwner: boolean;
  linkedInstallOnly: boolean;
}) {
  const { account } = useAuth();
  const [instances, setInstances] = useState<AssetInstance[]>([]);
  const [format, setFormat] = useState("");
  const [app, setApp] = useState(appTargets[0]?.id ?? "");
  const [openFormats, setOpenFormats] = useState(false);
  const [destination, setDestination] = useState(DOWNLOAD_DESTINATION);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const gallery = useMemo(
    () => travellingGallery({ blocks, images }),
    [blocks, images],
  );
  const [taken, setTaken] = useState<string[] | null>(null);
  const watched = useRef(0);

  const read = useCallback(async () => {
    const response = await fetch(`/api/v1/assets/${assetId}/instances`, {
      cache: "no-store",
      credentials: "same-origin",
    });
    if (!response.ok) {
      setInstances([]);
      return;
    }
    const answer = (await response.json()) as AssetInstanceList;
    setInstances(answer.items);
  }, [assetId]);

  useEffect(() => {
    if (!account) {
      setInstances([]);
      return;
    }
    void read();
  }, [account, read]);

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
  const pending = Boolean(
    instance?.delivery && instance.delivery.state !== "failed",
  );
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

  useEffect(() => {
    if (!pending) {
      watched.current = 0;
      return;
    }
    if (watched.current >= WATCH_LIMIT) return;
    const timer = setTimeout(() => {
      watched.current += 1;
      void read();
    }, WATCH_INTERVAL_MS);
    return () => clearTimeout(timer);
  }, [pending, read]);

  if (destinations.length === 0) return null;
  if (!linkedInstallOnly && choices.length === 0 && !original) return null;

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
      await read();
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
    <Popover onOpenChange={() => setFailure("")}>
      <PopoverTrigger asChild>
        <Button className="min-w-52 justify-between" variant="primary">
          Download {kindLabel}
          <ChevronDown aria-hidden="true" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" aria-label={`Download this ${kindLabel}`}>
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
              Send to
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
                href={downloadAddress({
                  assetId,
                  format: chosen?.format ?? "",
                  carried: carriesGallery,
                  gallery,
                  included,
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
              onClick={() =>
                act(`/api/v1/assets/${assetId}/deliveries`, "POST", {
                  instanceId: instance.instanceId,
                })
              }
              variant="primary"
            >
              {pending ? (
                <Clock aria-hidden="true" />
              ) : (
                <Send aria-hidden="true" />
              )}
              {sendActionLabel(instance)}
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

        {original && isOwner ? (
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
      </PopoverContent>
    </Popover>
  );
}

function downloadLabel(
  choice: FormatChoice | undefined,
  named: string,
): string {
  if (named) return `Download for ${named}`;
  return `Download ${choice?.label ?? ""}`;
}

/** downloadAddress names the reader's images only where they differ from the creator's. */
function downloadAddress({
  assetId,
  format,
  carried,
  gallery,
  included,
}: {
  assetId: string;
  format: string;
  carried: boolean;
  gallery: TravellingImage[];
  included: string[];
}): string {
  const address = `/download/${assetId}/${format}`;
  if (!carried) return address;
  const byDefault = gallery
    .filter((one) => one.chosen)
    .map((one) => one.mediaId);
  if (
    byDefault.length === included.length &&
    byDefault.every((mediaId) => included.includes(mediaId))
  ) {
    return address;
  }
  return `${address}?images=${included.join(",")}`;
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

/** WhatTravels says what reaches the reader, in the named app's terms where one is chosen. */
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

  if (delivery && delivery.state !== "failed") {
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
    <p className="mt-4 text-meta text-mute">{instanceStanding(instance)}</p>
  );
}
