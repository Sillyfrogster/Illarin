"use client";

import { History } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import type { RecordedVersion, WorkDetail } from "@/lib/api/query";
import { cn, focusRing } from "@/lib/cn";
import { versionDate, versionSummary } from "@/lib/work-versions";
import { VersionHistory } from "../history/VersionHistory";
import { listedVersions, useCardStage } from "./stage";
import { versionName } from "./WorkCard";

const RAIL_LENGTH = 12;

/** VersionTimeline is the index of the card deck: one point per version, pointing pulls that card up, choosing it brings it to the front. */
export function VersionTimeline({
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
  const stage = useCardStage();
  const latest = work.latestVersion;
  if (!latest) return null;
  const versions = listedVersions(stage.versions);
  const newest = versions[0] ?? latest;
  const shown = stage.viewing ?? newest;
  const count = versions.length || latest.number;
  const history = (
    <VersionHistory
      download={download}
      focus={shown.number}
      typeLabel={typeLabel}
      typeName={typeName}
      work={work}
    >
      <Button className="-mr-3" size="compact" variant="ghost">
        <History aria-hidden="true" />
        All versions
      </Button>
    </VersionHistory>
  );

  if (count <= 1) {
    return (
      <section className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
        <p className="text-meta text-mute">
          <span className="font-medium text-ink">{versionName(latest)}</span> ·{" "}
          {versionDate(latest)}
        </p>
        {history}
      </section>
    );
  }

  const rail = versions.slice(0, RAIL_LENGTH).reverse();
  const earlier = count - Math.min(count, RAIL_LENGTH);
  const waiting = Array.from(
    { length: Math.min(count, RAIL_LENGTH) },
    (_, at) => latest.number - Math.min(count, RAIL_LENGTH) + 1 + at,
  );

  return (
    <section aria-labelledby="versions-heading">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-ui font-medium text-ink" id="versions-heading">
          Versions <span className="font-normal text-mute">· {count}</span>
        </h2>
        {history}
      </div>

      <div className="mt-3 flex items-center gap-3">
        {earlier > 0 ? (
          <span className="shrink-0 text-meta text-mute">
            {earlier} earlier
          </span>
        ) : null}
        <ol
          aria-label="Versions, oldest first"
          className="relative flex min-h-8 min-w-0 flex-1 list-none items-center justify-between before:absolute before:inset-x-4 before:top-1/2 before:h-px before:bg-rule"
        >
          {rail.length > 0
            ? rail.map((version) => (
                <Point
                  here={version.number === shown.number}
                  key={version.number}
                  latest={version.number === newest.number}
                  version={version}
                />
              ))
            : waiting.map((number) => (
                <li className="grid size-8 place-items-center" key={number}>
                  <span
                    className={cn(
                      "block rounded-full",
                      number === latest.number
                        ? "size-3 bg-accent ring-4 ring-accent/20"
                        : "size-2 bg-mute",
                    )}
                  />
                </li>
              ))}
        </ol>
      </div>

      <article aria-live="polite" className="mt-4 flex flex-col gap-1">
        <p className="text-meta text-mute">
          <span className="font-medium text-ink">{versionName(shown)}</span> ·{" "}
          {versionDate(shown)}
          {shown.number === newest.number ? " · Latest" : null}
        </p>
        <p className="text-ui text-ink">{versionSummary(shown, typeName)}</p>
        {shown.notes ? (
          <p className="line-clamp-3 text-meta whitespace-pre-line text-mute">
            {shown.notes}
          </p>
        ) : null}
        {stage.viewing ? (
          <Button
            className="mt-2 self-start"
            onClick={() => stage.view(null)}
            size="compact"
          >
            Back to latest
          </Button>
        ) : null}
      </article>
    </section>
  );
}

function Point({
  here,
  latest,
  version,
}: {
  here: boolean;
  latest: boolean;
  version: RecordedVersion;
}) {
  const stage = useCardStage();
  const withdrawn = Boolean(version.withdrawnAt);
  const point = (show: boolean) =>
    stage.setPeek(show && !here ? version : null);
  return (
    <li className="relative">
      <button
        aria-current={here || undefined}
        aria-label={`${versionName(version)}, ${versionDate(version)}${version.summary ? `: ${version.summary}` : ""}`}
        className={cn(
          "group/point grid size-8 cursor-pointer place-items-center rounded-full",
          focusRing,
        )}
        onBlur={() => point(false)}
        onClick={() => stage.view(latest ? null : version)}
        onFocus={() => point(true)}
        onPointerEnter={(event) => event.pointerType === "mouse" && point(true)}
        onPointerLeave={() => point(false)}
        type="button"
      >
        <span
          className={cn(
            "block rounded-full transition-transform duration-200 ease-(--ease-wipe) group-hover/point:scale-150",
            here
              ? "size-3 bg-accent ring-4 ring-accent/20"
              : withdrawn
                ? "size-2 bg-field ring-1 ring-stop"
                : "size-2 bg-mute",
          )}
        />
      </button>
    </li>
  );
}
