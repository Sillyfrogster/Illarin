"use client";

import { ChevronDown, CircleAlert } from "lucide-react";
import { type ReactNode, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  fetchRecordedVersionDownloads,
  type RecordedVersion,
  type RecordedVersionDownloads,
} from "@/lib/api/query";
import { assetHoldsNothing } from "@/lib/asset-page-content";
import { versionTitle } from "@/lib/asset-updates";
import { AssetChooser } from "../AssetChooser";

type Reading =
  | { state: "unread" }
  | { state: "reading" }
  | { state: "read"; offered: RecordedVersionDownloads }
  | { state: "refused"; refusal: string; retry: boolean };

async function noRefresh(): Promise<void> {}

/** Loads a historical version's download choices when the chooser opens. */
export function VersionDownload({
  assetId,
  kind,
  version,
}: {
  assetId: string;
  kind: string;
  version: RecordedVersion;
}) {
  const [reading, setReading] = useState<Reading>({ state: "unread" });

  async function read() {
    setReading({ state: "reading" });
    const answer = await fetchRecordedVersionDownloads(assetId, version.number);
    setReading(
      answer.offered
        ? { state: "read", offered: answer.offered }
        : { state: "refused", refusal: answer.refusal, retry: answer.retry },
    );
  }

  return (
    <Popover
      onOpenChange={(open) => {
        if (open && reading.state === "unread") void read();
      }}
    >
      <PopoverTrigger asChild>
        <Button className="min-w-52 justify-between">
          Download this version
          <ChevronDown aria-hidden="true" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        aria-label={`Download ${versionTitle(version).toLowerCase()}`}
      >
        <VersionChoices
          assetId={assetId}
          kind={kind}
          onRetry={read}
          reading={reading}
          version={version}
        />
      </PopoverContent>
    </Popover>
  );
}

function VersionChoices({
  assetId,
  kind,
  version,
  reading,
  onRetry,
}: {
  assetId: string;
  kind: string;
  version: RecordedVersion;
  reading: Reading;
  onRetry: () => void;
}) {
  if (reading.state === "unread" || reading.state === "reading") {
    return (
      <p aria-busy="true" className="text-meta text-mute">
        Loading download options…
      </p>
    );
  }
  if (reading.state === "refused") {
    return (
      <Refusal onRetry={reading.retry ? onRetry : null}>
        {reading.refusal}
      </Refusal>
    );
  }
  const { offered } = reading;
  if (offered.linkedInstallOnly) {
    return (
      <Refusal onRetry={null}>
        File downloads are unavailable because this version contains protected
        prompt content.
      </Refusal>
    );
  }
  if (offered.downloads.length === 0) {
    return (
      <Refusal onRetry={null}>
        This version cannot be exported in any currently supported format.
      </Refusal>
    );
  }
  return (
    <AssetChooser
      appTargets={offered.appTargets}
      assetId={assetId}
      blocks={offered.blocks}
      downloads={offered.downloads}
      holdsNothing={assetHoldsNothing(offered.blocks)}
      images={offered.media}
      instances={[]}
      isOwner={false}
      kind={offered.kind}
      kindLabel={kind}
      linkedInstallOnly={false}
      original={null}
      refresh={noRefresh}
      version={version}
    />
  );
}

function Refusal({
  children,
  onRetry,
}: {
  children: ReactNode;
  onRetry: (() => void) | null;
}) {
  return (
    <div className="grid justify-items-start gap-3">
      <p className="flex items-start gap-2 text-meta text-mute">
        <CircleAlert aria-hidden="true" className="mt-0.5 size-3.5 shrink-0" />
        <span>{children}</span>
      </p>
      {onRetry ? (
        <Button onClick={onRetry} size="compact">
          Try again
        </Button>
      ) : null}
    </div>
  );
}
