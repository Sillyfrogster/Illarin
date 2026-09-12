"use client";

import { CircleSlash2, PencilLine, RotateCcw } from "lucide-react";
import { useRouter } from "next/navigation";
import { type ReactNode, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { Select } from "@/components/ui/select";
import {
  correctAssetVersionNotes,
  type RecordedVersion,
  restoreAssetVersion,
  withdrawAssetVersion,
} from "@/lib/api/query";
import {
  earlierVersions,
  isLongNote,
  versionAnchor,
  versionDate,
  versionSummary,
  versionTitle,
} from "@/lib/asset-updates";
import { assetHref } from "@/lib/asset-url";
import { cn } from "@/lib/cn";
import type { Candidate } from "@/lib/working-copy";
import { VersionChanges } from "./VersionChanges";

export type HistoryOwner = {
  assetName: string;
  canManage: boolean;
  isOwner: boolean;
  workingCopyVersion: number;
};

export function VersionEntry({
  assetId,
  kind,
  version,
  versions,
  current,
  download,
  owner,
}: {
  assetId: string;
  kind: string;
  version: RecordedVersion;
  versions: RecordedVersion[];
  current: boolean;
  download: ReactNode;
  owner: HistoryOwner;
}) {
  const earlier = earlierVersions(versions, version);
  const [baseline, setBaseline] = useState(earlier[0]?.number ?? 0);
  const against = earlier.find((one) => one.number === baseline);
  const anchor = versionAnchor(version);
  const withdrawn = Boolean(
    version.withdrawnAt || version.withdrawalExplanation,
  );

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
            Published
          </span>
        ) : null}
      </div>

      <p className="mt-1.5 flex flex-wrap items-baseline gap-x-3 gap-y-1 text-meta text-mute">
        <time dateTime={version.recordedAt}>{versionDate(version)}</time>
        {version.versionLabel ? (
          <span>Creator’s version {version.versionLabel}</span>
        ) : null}
        {version.notesEditedAt ? <span>Notes edited</span> : null}
        {withdrawn ? (
          <span className="font-medium text-stop">Withdrawn</span>
        ) : null}
      </p>

      {withdrawn && !owner.isOwner ? (
        <p className="mt-4 max-w-[60ch] font-prose text-prose text-mute">
          {version.withdrawalExplanation}
        </p>
      ) : (
        <>
          <p className="mt-4 max-w-[60ch] font-prose text-prose text-ink">
            {versionSummary(version, kind)}
          </p>

          {version.notes ? <Note notes={version.notes} /> : null}
        </>
      )}

      <div className="mt-4">
        {withdrawn && !owner.isOwner ? (
          <p className="text-meta text-mute">
            Its content, comparisons and downloads are unavailable.
          </p>
        ) : against ? (
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

      {owner.canManage ? (
        <VersionManagement
          assetId={assetId}
          current={current}
          owner={owner}
          version={version}
        />
      ) : null}
    </li>
  );
}

function VersionManagement({
  assetId,
  current,
  owner,
  version,
}: {
  assetId: string;
  current: boolean;
  owner: HistoryOwner;
  version: RecordedVersion;
}) {
  const router = useRouter();
  const [mode, setMode] = useState<"" | "restore" | "correct" | "withdraw">("");
  const [summary, setSummary] = useState(version.summary);
  const [notes, setNotes] = useState(version.notes);
  const [explanation, setExplanation] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    setSummary(version.summary);
    setNotes(version.notes);
  }, [version.notes, version.summary]);

  async function runMutation(work: () => Promise<void>, done: () => void) {
    setBusy(true);
    setMessage("");
    try {
      await work();
      done();
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "That change could not be saved.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-6 max-w-[38rem] border-rule border-t pt-2">
      <div className="flex min-h-11 flex-wrap items-center gap-x-1">
        {!current ? (
          <Button
            aria-pressed={mode === "restore"}
            className={cn("-ml-3", mode === "restore" && "text-ink")}
            onClick={() => setMode("restore")}
            size="compact"
            variant="ghost"
          >
            <RotateCcw aria-hidden="true" />
            Restore
          </Button>
        ) : null}
        <Button
          aria-pressed={mode === "correct"}
          className={cn(mode === "correct" && "text-ink")}
          onClick={() => setMode("correct")}
          size="compact"
          variant="ghost"
        >
          <PencilLine aria-hidden="true" />
          Edit notes
        </Button>
        {!current && !version.withdrawnAt ? (
          <Button
            aria-pressed={mode === "withdraw"}
            className={cn(mode === "withdraw" && "text-stop")}
            onClick={() => setMode("withdraw")}
            size="compact"
            variant="ghost"
          >
            <CircleSlash2 aria-hidden="true" />
            Withdraw
          </Button>
        ) : null}
      </div>

      {mode === "restore" ? (
        <div className="grid gap-4 pt-4 pb-1">
          <p className="text-meta text-mute">
            This replaces the private working copy with this version, including
            its pictures and page arrangement. Access, protection and delivery
            choices stay current. Publishing it later requires fresh update
            notes and validation.
          </p>
          <ActionRow
            busy={busy}
            confirm="Replace working copy"
            onCancel={() => setMode("")}
            onConfirm={() =>
              runMutation(
                () =>
                  restoreAssetVersion(
                    { version: owner.workingCopyVersion } satisfies Candidate,
                    assetId,
                    version.number,
                  ),
                () =>
                  window.location.assign(assetHref(assetId, owner.assetName)),
              )
            }
          />
        </div>
      ) : null}

      {mode === "correct" ? (
        <div className="grid gap-4 pt-4 pb-1">
          <label className="grid gap-1 text-meta text-mute">
            Summary
            <input
              className="min-h-11 rounded-control bg-field px-3 text-ui text-ink"
              maxLength={200}
              onChange={(event) => setSummary(event.target.value)}
              value={summary}
            />
          </label>
          <label className="grid gap-1 text-meta text-mute">
            Notes
            <textarea
              className="min-h-28 rounded-control bg-field p-3 text-ui text-ink"
              maxLength={4000}
              onChange={(event) => setNotes(event.target.value)}
              value={notes}
            />
          </label>
          <ActionRow
            busy={busy}
            confirm="Save correction"
            onCancel={() => setMode("")}
            onConfirm={() =>
              runMutation(
                () =>
                  correctAssetVersionNotes(assetId, version.number, {
                    summary,
                    notes,
                  }),
                () => {
                  setMode("");
                  router.refresh();
                },
              )
            }
          />
        </div>
      ) : null}

      {mode === "withdraw" ? (
        <div className="grid gap-4 pt-4 pb-1">
          <p className="text-meta text-mute">
            Readers will keep the version number, date and this explanation. Its
            content, comparisons and downloads will be blocked.
          </p>
          <label className="grid gap-1 text-meta text-mute">
            Public explanation
            <textarea
              className="min-h-24 rounded-control bg-field p-3 text-ui text-ink"
              maxLength={1000}
              onChange={(event) => setExplanation(event.target.value)}
              value={explanation}
            />
          </label>
          <ActionRow
            busy={busy}
            confirm="Withdraw version"
            onCancel={() => setMode("")}
            onConfirm={() =>
              runMutation(
                () =>
                  withdrawAssetVersion(assetId, version.number, explanation),
                () => {
                  setMode("");
                  router.refresh();
                },
              )
            }
            tone="stop"
          />
        </div>
      ) : null}

      {message ? (
        <p className="mt-3 text-meta text-stop" role="alert">
          {message}
        </p>
      ) : null}
    </div>
  );
}

function ActionRow({
  busy,
  confirm,
  onCancel,
  onConfirm,
  tone = "primary",
}: {
  busy: boolean;
  confirm: string;
  onCancel: () => void;
  onConfirm: () => void;
  tone?: "primary" | "stop";
}) {
  return (
    <div className="flex flex-wrap gap-2">
      <Button loading={busy} onClick={onConfirm} size="compact" variant={tone}>
        {confirm}
      </Button>
      <Button disabled={busy} onClick={onCancel} size="compact" variant="ghost">
        Cancel
      </Button>
    </div>
  );
}

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
          {shown ? "Show less" : "Read full notes"}
        </button>
      ) : null}
    </div>
  );
}
