"use client";

import { ChevronDown, CircleAlert } from "lucide-react";
import { useState } from "react";
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
import { downloadAddress, orderedFormats } from "@/lib/work-send";
import { versionDate, versionTitle } from "@/lib/work-versions";

type Reading =
  | { state: "unread" }
  | { state: "reading" }
  | { state: "read"; offered: RecordedVersionDownloads }
  | { state: "refused"; refusal: string; retry: boolean };

/** Loads a historical version's formats when the chooser opens. */
export function VersionDownload({
  workId,
  version,
}: {
  workId: string;
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
        className="p-3"
      >
        <VersionChoices
          workId={workId}
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
  version,
  reading,
  onRetry,
}: {
  workId: string;
  version: RecordedVersion;
  reading: Reading;
  onRetry: () => void;
}) {
  if (reading.state === "unread" || reading.state === "reading") {
    return (
      <p aria-busy="true" className="p-2 text-meta text-mute">
        Loading the formats…
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
        This version has private prompts, so it does not download as a file.
      </Refusal>
    );
  }
  if (offered.downloads.length === 0) {
    return (
      <Refusal onRetry={null}>
        This version cannot be written in any format Illarin offers.
      </Refusal>
    );
  }
  return (
    <>
      <p className="max-w-[42ch] px-2 pt-1 pb-2 text-meta text-mute">
        Written now from what this version recorded on {versionDate(version)},
        not the file uploaded then.
      </p>
      <ul className="flex list-none flex-col">
        {orderedFormats(offered.downloads).map((one) => (
          <li key={one.format}>
            <a
              className="flex min-h-11 items-center rounded-control px-3 text-ui text-ink outline-offset-3 hover:bg-accent-wash"
              href={downloadAddress({
                workId,
                format: one.format,
                version: version.number,
              })}
            >
              {one.label}
            </a>
          </li>
        ))}
      </ul>
    </>
  );
}

function Refusal({
  children,
  onRetry,
}: {
  children: React.ReactNode;
  onRetry: (() => void) | null;
}) {
  return (
    <div className="grid justify-items-start gap-3 p-2">
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
