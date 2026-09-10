"use client";

import {
  ChevronDown,
  CircleAlert,
  Clock,
  Download,
  FileDown,
  Send,
} from "lucide-react";
import Image from "next/image";
import { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Select } from "@/components/ui/select";
import { browserFetch } from "@/lib/api/browser-mutation";
import type {
  AssetImage,
  AssetInstance,
  AssetInstanceList,
  DownloadTarget,
  OriginalUpload,
} from "@/lib/api/query";
import {
  DOWNLOAD_DESTINATION,
  deliveryDestinations,
  deliveryFailureLine,
  type FormatChoice,
  formatChoices,
  instanceStanding,
  itemCount,
  sendActionLabel,
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
  downloads,
  original,
  images,
  holdsNothing,
  isOwner,
  linkedInstallOnly,
}: {
  assetId: string;
  kindLabel: string;
  downloads: DownloadTarget[];
  original: OriginalUpload | null;
  images: AssetImage[];
  holdsNothing: boolean;
  isOwner: boolean;
  linkedInstallOnly: boolean;
}) {
  const { account } = useAuth();
  const [instances, setInstances] = useState<AssetInstance[]>([]);
  const [format, setFormat] = useState("");
  const [destination, setDestination] = useState(DOWNLOAD_DESTINATION);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
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

  const choices = formatChoices({ downloads, holdsNothing });
  const chosen =
    choices.find((choice) => choice.format === format) ?? choices[0];
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
          Get {kindLabel}
          <ChevronDown aria-hidden="true" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" aria-label={`Get this ${kindLabel}`}>
        {choices.length > 0 && !linkedInstallOnly ? (
          <fieldset className="min-w-0 border-0 p-0">
            <legend className="mb-2 text-meta font-medium text-ink">
              Format
            </legend>
            <div className="space-y-0.5">
              {choices.map((choice) => (
                <FormatRow
                  chosen={choice.format === chosen?.format}
                  choice={choice}
                  images={images}
                  key={choice.format}
                  onChoose={() => setFormat(choice.format)}
                />
              ))}
            </div>
          </fieldset>
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

        {toAFile ? (
          <Button asChild className="mt-4 w-full" variant="primary">
            <a href={`/download/${assetId}/${chosen?.format}`}>
              <Download aria-hidden="true" />
              Download {chosen?.label}
            </a>
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

        {original && isOwner ? (
          <div className="mt-5 border-rule border-t pt-4">
            <p className="text-meta font-medium text-ink">
              The creator’s own file
            </p>
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

function FormatRow({
  choice,
  chosen,
  images,
  onChoose,
}: {
  choice: FormatChoice;
  chosen: boolean;
  images: AssetImage[];
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
          <span className="block text-meta text-mute">
            {choice.recommended ? "Recommended · " : ""}
            {choice.cost}
          </span>
        </span>
      </label>
      {chosen && choice.losses.length > 0 ? (
        <ul className="space-y-3 px-3 pb-3">
          {choice.losses.map((loss) => (
            <li key={loss.role}>
              <p className="text-meta font-medium text-ink">
                {loss.label}
                <span className="font-normal text-mute">
                  {" "}
                  — {itemCount(loss.sample.count)}
                </span>
              </p>
              <p className="text-meta text-mute">{loss.line}</p>
              <Sample images={images} sample={loss.sample} />
            </li>
          ))}
        </ul>
      ) : null}
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
