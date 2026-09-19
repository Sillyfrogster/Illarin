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
  fetchWorkUpdates,
  type RecordedVersion,
  type WorkDetail,
} from "@/lib/api/query";
import { PHONE_WIDTH, useMediaQuery } from "@/lib/use-media-query";
import { workHref } from "@/lib/work-url";
import { versionDate, versionSummary, versionTitle } from "@/lib/work-versions";
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

/** Opens the latest version into the full version history. */
export function VersionHistory({
  work,
  download,
  typeName,
  typeLabel,
}: {
  work: WorkDetail;
  download: ReactNode;
  typeName: string;
  typeLabel: string;
}) {
  const router = useRouter();
  const phone = useMediaQuery(PHONE_WIDTH);
  const latest = work.latestVersion;
  const [open, setOpen] = useState(false);
  const [chosen, setChosen] = useState<number | null>(null);
  const [versions, setVersions] = useState<Versions>({ state: "unread" });

  const read = useCallback(async () => {
    setVersions({ state: "reading" });
    const items = await fetchWorkUpdates(work.id);
    setVersions(
      items ? { state: "read", versions: items } : { state: "refused" },
    );
  }, [work.id]);

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
      router.replace(workHref(work.id, work.name), { scroll: false });
    }
  }

  if (!latest) return null;

  const listed = versions.state === "read" ? versions.versions : [];
  const shown =
    chosen === null && !phone
      ? listed[0]
      : listed.find((v) => v.number === chosen);
  const owner: HistoryOwner = {
    workName: work.name,
    canManage: Boolean(work.isOwner && !work.withhold),
    isOwner: work.isOwner,
    draftedChangesVersion: work.draftedChangesVersion ?? 0,
  };
  const published = work.lifecycle !== "draft";

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
              {versionSummary(latest, typeName)}
            </span>
          </span>
          <span className="grid size-11 shrink-0 place-items-center rounded-full bg-plane text-accent transition-colors duration-200 group-hover:bg-action group-hover:text-on-accent motion-reduce:transition-none">
            <History aria-hidden="true" className="size-5" />
            <span className="sr-only">Open the version history</span>
          </span>
        </span>
      </ExpandingPanelTrigger>

      <ExpandingPanelContent>
        <header className="flex items-center gap-4 border-b border-rule px-4 py-3 sm:px-6">
          <span className="relative size-11 shrink-0 overflow-hidden rounded-control bg-deep">
            {work.media.find((image) => image.isCover) ? (
              <Image
                alt=""
                className="size-full object-cover"
                height={88}
                src={work.media.find((image) => image.isCover)?.thumbUrl ?? ""}
                unoptimized
                width={88}
              />
            ) : (
              <DefaultCover compact type={work.type} />
            )}
          </span>
          <div className="min-w-0 flex-1">
            <ExpandingPanelTitle className="font-display text-section font-medium text-ink">
              Version history
            </ExpandingPanelTitle>
            <p className="truncate font-ui text-meta text-mute">
              {work.name} · {typeLabel} by {work.creator}
            </p>
          </div>
          <ExpandingPanelClose label="Close the version history" />
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
            <HistoryVersions
              typeName={typeName}
              onRetry={read}
              versions={versions}
            >
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
                workId={work.id}
                current={shown.number === listed[0]?.number}
                download={
                  shown.number === listed[0]?.number ? (
                    published ? (
                      download
                    ) : null
                  ) : published && !work.hasPrivatePrompts ? (
                    <VersionDownload
                      workId={work.id}
                      typeName={typeName}
                      version={shown}
                    />
                  ) : null
                }
                key={shown.number}
                typeName={typeName}
                onChanged={() => void read()}
                owner={owner}
                version={shown}
                versions={listed}
              />
            ) : null}
            {shown && work.hasPrivatePrompts && published ? (
              <p className="mt-4 max-w-[60ch] font-ui text-meta text-mute">
                This {typeName} has private prompts, so no version downloads as
                a file. Only an allowed app can use it:{" "}
                {work.allowedApps.map((app) => app.label).join(", ")}.
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
  typeName,
  onRetry,
  versions,
}: {
  children: ReactNode;
  typeName: string;
  onRetry: () => void;
  versions: Versions;
}) {
  if (versions.state === "read") return children;
  if (versions.state === "refused") {
    return (
      <div className="grid justify-items-start gap-3 px-3 py-2">
        <p className="font-ui text-meta text-mute" role="alert">
          Illarin could not read the versions of this {typeName}.
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
