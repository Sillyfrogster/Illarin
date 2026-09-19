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
import { workHoldsNothing } from "@/lib/work-page-content";
import { versionTitle } from "@/lib/work-versions";
import { WorkChooser } from "../WorkChooser";

type Reading =
  | { state: "unread" }
  | { state: "reading" }
  | { state: "read"; offered: RecordedVersionDownloads }
  | { state: "refused"; refusal: string; retry: boolean };

async function noRefresh(): Promise<void> {}

/** Loads a historical version's download choices when the chooser opens. */
export function VersionDownload({
  workId,
  typeName,
  version,
}: {
  workId: string;
  typeName: string;
  version: RecordedVersion;
}) {
  const [reading, setReading] = useState<Reading>({ state: "unread" });

  async function read() {
    setReading({ state: "reading" });
    const answer = await fetchRecordedVersionDownloads(workId, version.number);
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
          workId={workId}
          typeName={typeName}
          onRetry={read}
          reading={reading}
          version={version}
        />
      </PopoverContent>
    </Popover>
  );
}

function VersionChoices({
  workId,
  typeName,
  version,
  reading,
  onRetry,
}: {
  workId: string;
  typeName: string;
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
  if (offered.hasPrivatePrompts) {
    return (
      <Refusal onRetry={null}>
        File downloads are unavailable because this version has private prompts.
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
    <WorkChooser
      appFormats={offered.appFormats}
      workId={workId}
      blocks={offered.blocks}
      downloads={offered.downloads}
      holdsNothing={workHoldsNothing(offered.blocks)}
      images={offered.media}
      connectedApps={[]}
      isOwner={false}
      type={offered.type}
      typeLabel={typeName}
      hasPrivatePrompts={false}
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
