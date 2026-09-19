"use client";

import {
  Timeline,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineItem,
} from "@/components/ui/timeline";
import type { RecordedVersion } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { versionDate, versionTitle } from "@/lib/work-versions";

/** VersionList runs every recorded version down one line, newest first. */
export function VersionList({
  chosen,
  onChoose,
  versions,
}: {
  chosen: number | null;
  onChoose: (number: number) => void;
  versions: RecordedVersion[];
}) {
  return (
    <Timeline
      aria-label="Recorded versions"
      className="[--timeline-dot-offset:0.9375rem]"
    >
      {versions.map((version, index) => {
        const withdrawn = Boolean(
          version.withdrawnAt || version.withdrawalExplanation,
        );
        const here = version.number === chosen;
        return (
          <TimelineItem key={version.id}>
            <TimelineDot
              className={cn(
                index === 0 && "border-accent bg-accent",
                withdrawn && "border-stop",
              )}
            />
            <TimelineConnector />
            <TimelineContent asChild>
              <button
                aria-current={here ? "true" : undefined}
                className={cn(
                  "flex min-h-11 w-full flex-col items-start rounded-control px-3 py-2 text-left outline-offset-2 transition-colors duration-150 motion-reduce:transition-none",
                  here ? "bg-accent-wash" : "hover:bg-deep",
                )}
                onClick={() => onChoose(version.number)}
                type="button"
              >
                <span className="flex flex-wrap items-baseline gap-x-2 font-ui text-ui font-medium text-ink">
                  {versionTitle(version)}
                  {index === 0 ? (
                    <span className="text-label font-medium text-accent">
                      Published
                    </span>
                  ) : null}
                  {withdrawn ? (
                    <span className="text-label font-medium text-stop">
                      Withdrawn
                    </span>
                  ) : null}
                </span>
                <span className="font-ui text-meta text-mute">
                  <time dateTime={version.recordedAt}>
                    {versionDate(version)}
                  </time>
                  {version.versionLabel ? ` · v${version.versionLabel}` : null}
                </span>
              </button>
            </TimelineContent>
          </TimelineItem>
        );
      })}
    </Timeline>
  );
}
