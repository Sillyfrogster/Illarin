"use client";

import { Badge } from "@/components/ui/badge";
import { Item, ItemGroup } from "@/components/ui/item";
import type { RecordedVersion } from "@/lib/api/query";
import { cn, focusRing } from "@/lib/cn";
import { versionTag } from "@/lib/version-label";
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
    <ItemGroup
      as="ol"
      className="-mx-3 bg-transparent"
      label="Recorded versions"
    >
      {versions.map((version, index) => {
        const withdrawn = Boolean(
          version.withdrawnAt || version.withdrawalExplanation,
        );
        const here = version.number === chosen;
        return (
          <Item className="p-0 not-first:border-t-0" key={version.id}>
            <button
              aria-current={here ? "true" : undefined}
              className={cn(
                "flex min-h-control w-full cursor-pointer flex-col items-start rounded-control px-3 py-2 text-left",
                focusRing,
                here && "bg-accent-wash",
              )}
              onClick={() => onChoose(version.number)}
              type="button"
            >
              <span
                className={cn(
                  "flex flex-wrap items-center gap-x-2 font-ui text-ui font-medium",
                  here ? "text-accent" : "text-ink",
                )}
              >
                {versionTitle(version)}
                {index === 0 ? (
                  <Badge size="compact" tone="accent">
                    Published
                  </Badge>
                ) : null}
                {withdrawn ? (
                  <Badge size="compact" tone="stop">
                    Withdrawn
                  </Badge>
                ) : null}
              </span>
              <span className="font-ui text-meta text-mute">
                <time dateTime={version.recordedAt}>
                  {versionDate(version)}
                </time>
                {version.versionLabel
                  ? ` · ${versionTag(version.versionLabel)}`
                  : null}
              </span>
            </button>
          </Item>
        );
      })}
    </ItemGroup>
  );
}
