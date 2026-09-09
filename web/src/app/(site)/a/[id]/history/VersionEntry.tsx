"use client";

import { type ReactNode, useState } from "react";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { Select } from "@/components/ui/select";
import type { RecordedVersion } from "@/lib/api/query";
import {
  earlierVersions,
  isLongNote,
  versionAnchor,
  versionDate,
  versionSummary,
  versionTitle,
} from "@/lib/asset-updates";
import { cn } from "@/lib/cn";
import { VersionChanges } from "./VersionChanges";

/** One recorded version, what Illarin computed for it, and how to take it. */
export function VersionEntry({
  assetId,
  kind,
  version,
  versions,
  current,
  download,
}: {
  assetId: string;
  kind: string;
  version: RecordedVersion;
  versions: RecordedVersion[];
  /** Whether this is the version readers get when they take the asset now. */
  current: boolean;
  download: ReactNode;
}) {
  const earlier = earlierVersions(versions, version);
  const [baseline, setBaseline] = useState(earlier[0]?.number ?? 0);
  const against = earlier.find((one) => one.number === baseline);
  const anchor = versionAnchor(version);

  return (
    <li
      className="relative scroll-mt-[calc(var(--header-height)+2rem)] border-rule pb-section pl-7 last:border-transparent last:pb-0 sm:pl-9 [&:not(:last-child)]:border-l"
      id={anchor}
    >
      <span
        aria-hidden="true"
        className={cn(
          "absolute top-1.5 -left-[5px] size-2.5 rounded-full",
          current ? "bg-accent" : "bg-edge",
        )}
      />

      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h2 className="font-display text-section font-medium text-ink">
          {versionTitle(version)}
        </h2>
        {current ? (
          <span className="rounded-control bg-accent-wash px-2 py-0.5 text-meta font-medium text-accent">
            Readers have this
          </span>
        ) : null}
      </div>

      <p className="mt-1.5 flex flex-wrap items-baseline gap-x-3 gap-y-1 text-meta text-mute">
        <time dateTime={version.recordedAt}>{versionDate(version)}</time>
        {version.versionLabel ? (
          <span>Creator’s version {version.versionLabel}</span>
        ) : null}
      </p>

      <p className="mt-4 max-w-[60ch] font-prose text-prose text-ink">
        {versionSummary(version, kind)}
      </p>

      {version.notes ? <Note notes={version.notes} /> : null}

      <div className="mt-4">
        {against ? (
          <MorphingDisclosure
            summary={`What changed since ${versionTitle(against).toLowerCase()}`}
          >
            <Baseline
              chosen={baseline}
              earlier={earlier}
              onChoose={setBaseline}
              version={version}
            />
            <VersionChanges
              assetId={assetId}
              from={baseline}
              kind={kind}
              to={version.number}
            />
          </MorphingDisclosure>
        ) : (
          <p className="text-meta text-mute">
            Illarin recorded nothing before this, so there is nothing to compare
            it with.
          </p>
        )}
      </div>

      {download ? (
        <div className="mt-5 grid justify-items-start gap-3">{download}</div>
      ) : null}
    </li>
  );
}

/** The version this one is measured against, which a reader may move to any earlier one. */
function Baseline({
  earlier,
  chosen,
  version,
  onChoose,
}: {
  earlier: RecordedVersion[];
  chosen: number;
  version: RecordedVersion;
  onChoose: (number: number) => void;
}) {
  const field = `baseline-${version.number}`;

  return (
    <p className="mt-3 flex flex-wrap items-center gap-2 text-meta text-mute">
      <label htmlFor={field}>Compared with</label>
      <Select
        className="text-meta"
        id={field}
        onChange={(event) => onChoose(Number(event.target.value))}
        value={chosen}
      >
        {earlier.map((one) => (
          <option key={one.id} value={one.number}>
            {versionTitle(one)} · {versionDate(one)}
          </option>
        ))}
      </Select>
    </p>
  );
}

/** A creator's longer note, folded until a reader asks for the rest of it. */
function Note({ notes }: { notes: string }) {
  const foldable = isLongNote(notes);
  const [shown, setShown] = useState(!foldable);

  return (
    <div className="mt-3 max-w-[60ch]">
      <p
        className={cn(
          "font-prose text-prose whitespace-pre-wrap text-mute",
          shown ? null : "line-clamp-4",
        )}
      >
        {notes}
      </p>
      {foldable ? (
        <button
          aria-expanded={shown}
          className="mt-1 inline-flex min-h-11 items-center text-meta font-medium text-accent outline-offset-3 hover:text-ink"
          onClick={() => setShown(!shown)}
          type="button"
        >
          {shown ? "Fold the note" : "Read the whole note"}
        </button>
      ) : null}
    </div>
  );
}
