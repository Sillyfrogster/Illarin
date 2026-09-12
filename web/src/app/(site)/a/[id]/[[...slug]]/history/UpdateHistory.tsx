"use client";

import { ArrowLeft, History } from "lucide-react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { type ReactNode, useCallback, useEffect, useState } from "react";
import { DefaultCover } from "@/components/media/DefaultCover";
import { Button } from "@/components/ui/button";
import {
  ExpandingPanel,
  ExpandingPanelClose,
  ExpandingPanelContent,
  ExpandingPanelTitle,
  ExpandingPanelTrigger,
} from "@/components/ui/expanding-panel";
import {
  type AssetDetail,
  fetchAssetUpdates,
  type RecordedVersion,
} from "@/lib/api/query";
import { versionDate, versionSummary, versionTitle } from "@/lib/asset-updates";
import { assetHref } from "@/lib/asset-url";
import { protectedAppLabel } from "@/lib/protected-apps";
import { PHONE_WIDTH, useMediaQuery } from "@/lib/use-media-query";
import { type HistoryOwner, VersionDetail } from "./VersionDetail";
import { VersionDownload } from "./VersionDownload";
import { VersionList } from "./VersionList";

const HISTORY_QUERY = "history";

type Versions =
  | { state: "unread" }
  | { state: "reading" }
  | { state: "read"; versions: RecordedVersion[] }
  | { state: "refused" };

function versionInAddress(): number | null {
  const found = /^#version-(\d+)$/.exec(window.location.hash);
  return found ? Number(found[1]) : null;
}

/** Opens the latest update into the full version history. */
export function UpdateHistory({
  asset,
  download,
  kind,
  kindLabel,
}: {
  asset: AssetDetail;
  download: ReactNode;
  kind: string;
  kindLabel: string;
}) {
  const router = useRouter();
  const phone = useMediaQuery(PHONE_WIDTH);
  const latest = asset.latestUpdate;
  const [open, setOpen] = useState(false);
  const [chosen, setChosen] = useState<number | null>(null);
  const [versions, setVersions] = useState<Versions>({ state: "unread" });

  const read = useCallback(async () => {
    setVersions({ state: "reading" });
    const items = await fetchAssetUpdates(asset.id);
    setVersions(
      items ? { state: "read", versions: items } : { state: "refused" },
    );
  }, [asset.id]);

  useEffect(() => {
    if (!new URLSearchParams(window.location.search).has(HISTORY_QUERY)) return;
    setChosen(versionInAddress());
    setOpen(true);
  }, []);

  useEffect(() => {
    if (open && versions.state === "unread") void read();
  }, [open, read, versions.state]);

  function change(next: boolean) {
    setOpen(next);
    if (next) return;
    setChosen(null);
    if (new URLSearchParams(window.location.search).has(HISTORY_QUERY)) {
      router.replace(assetHref(asset.id, asset.name), { scroll: false });
    }
  }

  if (!latest) return null;

  const listed = versions.state === "read" ? versions.versions : [];
  const shown =
    chosen === null && !phone
      ? listed[0]
      : listed.find((v) => v.number === chosen);
  const owner: HistoryOwner = {
    assetName: asset.name,
    canManage: Boolean(asset.isOwner && !asset.withhold),
    isOwner: asset.isOwner,
    workingCopyVersion: asset.workingCopyVersion ?? 0,
  };
  const published = asset.lifecycle !== "draft";

  return (
    <ExpandingPanel onOpenChange={change} open={open}>
      <ExpandingPanelTrigger className="mt-8 max-w-[42ch]">
        <span className="flex items-center gap-4 py-4 pr-4 pl-5">
          <span className="flex min-w-0 flex-1 flex-col gap-1">
            <span className="flex flex-wrap items-baseline gap-x-3 gap-y-1 font-ui text-meta text-mute">
              <span className="font-medium text-ink">
                {versionTitle(latest)}
              </span>
              <time dateTime={latest.recordedAt}>{versionDate(latest)}</time>
              {latest.versionLabel ? (
                <span>Creator’s version {latest.versionLabel}</span>
              ) : null}
            </span>
            <span className="line-clamp-2 font-ui text-meta text-mute">
              {versionSummary(latest, kind)}
            </span>
          </span>
          <span className="grid size-11 shrink-0 place-items-center rounded-full bg-plane text-accent transition-colors duration-200 group-hover:bg-action group-hover:text-on-accent motion-reduce:transition-none">
            <History aria-hidden="true" className="size-5" />
            <span className="sr-only">Open the update history</span>
          </span>
        </span>
      </ExpandingPanelTrigger>

      <ExpandingPanelContent>
        <header className="flex items-center gap-4 border-b border-rule px-4 py-3 sm:px-6">
          <span className="relative size-11 shrink-0 overflow-hidden rounded-control bg-deep">
            {asset.media.find((image) => image.isCover) ? (
              <Image
                alt=""
                className="size-full object-cover"
                height={88}
                src={asset.media.find((image) => image.isCover)?.thumbUrl ?? ""}
                unoptimized
                width={88}
              />
            ) : (
              <DefaultCover compact kind={asset.kind} />
            )}
          </span>
          <div className="min-w-0 flex-1">
            <ExpandingPanelTitle className="font-display text-section font-medium text-ink">
              Update history
            </ExpandingPanelTitle>
            <p className="truncate font-ui text-meta text-mute">
              {asset.name} · {kindLabel} by {asset.creator}
            </p>
          </div>
          <ExpandingPanelClose label="Close the update history" />
        </header>

        <div className="grid min-h-0 flex-1 md:grid-cols-[18rem_minmax(0,1fr)]">
          <nav
            aria-label="Recorded versions"
            className={
              phone && shown
                ? "hidden"
                : "min-h-0 overflow-y-auto border-rule bg-inset px-3 py-5 sm:px-4 md:border-r"
            }
          >
            <HistoryVersions kind={kind} onRetry={read} versions={versions}>
              <VersionList
                chosen={shown?.number ?? null}
                onChoose={setChosen}
                versions={listed}
              />
            </HistoryVersions>
          </nav>
          <section
            className={
              phone && !shown
                ? "hidden"
                : "min-h-0 overflow-y-auto px-4 py-6 sm:px-8 sm:py-8"
            }
          >
            {phone && shown ? (
              <Button
                className="-ml-3 mb-4"
                onClick={() => setChosen(null)}
                size="compact"
                variant="ghost"
              >
                <ArrowLeft aria-hidden="true" />
                All versions
              </Button>
            ) : null}
            {shown ? (
              <VersionDetail
                assetId={asset.id}
                current={shown.number === listed[0]?.number}
                download={
                  shown.number === listed[0]?.number ? (
                    published ? (
                      download
                    ) : null
                  ) : published && !asset.linkedInstallOnly ? (
                    <VersionDownload
                      assetId={asset.id}
                      kind={kind}
                      version={shown}
                    />
                  ) : null
                }
                key={shown.number}
                kind={kind}
                onChanged={() => void read()}
                owner={owner}
                version={shown}
                versions={listed}
              />
            ) : null}
            {shown && asset.linkedInstallOnly && published ? (
              <p className="mt-4 max-w-[60ch] font-ui text-meta text-mute">
                This {kind} can only be installed through a linked app. File
                downloads are unavailable for all versions. Allowed apps:{" "}
                {asset.allowedApps.map(protectedAppLabel).join(", ")}.
              </p>
            ) : null}
          </section>
        </div>
      </ExpandingPanelContent>
    </ExpandingPanel>
  );
}

function HistoryVersions({
  children,
  kind,
  onRetry,
  versions,
}: {
  children: ReactNode;
  kind: string;
  onRetry: () => void;
  versions: Versions;
}) {
  if (versions.state === "read") return children;
  if (versions.state === "refused") {
    return (
      <div className="grid justify-items-start gap-3 px-3 py-2">
        <p className="font-ui text-meta text-mute" role="alert">
          Illarin could not read the versions of this {kind}.
        </p>
        <Button onClick={onRetry} size="compact">
          Try again
        </Button>
      </div>
    );
  }
  return (
    <output aria-label="Reading the versions" className="grid gap-4 px-3 py-2">
      {[0, 1, 2].map((slot) => (
        <span className="grid gap-2" key={slot}>
          <span className="h-3.5 w-28 animate-pulse rounded-control bg-deep motion-reduce:animate-none" />
          <span className="h-3 w-40 animate-pulse rounded-control bg-deep motion-reduce:animate-none" />
        </span>
      ))}
    </output>
  );
}
