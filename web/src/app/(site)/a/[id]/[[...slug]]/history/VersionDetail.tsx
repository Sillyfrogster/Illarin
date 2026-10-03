"use client";

import {
  ChevronRight,
  CircleSlash2,
  PencilLine,
  RotateCcw,
} from "lucide-react";
import { type ReactNode, useEffect, useId, useState } from "react";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Field } from "@/components/ui/field";
import { Input, Textarea } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import {
  correctWorkVersionNotes,
  type RecordedVersion,
  restoreWorkVersion,
  withdrawWorkVersion,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import type { Candidate } from "@/lib/drafted-changes";
import { versionTag } from "@/lib/version-label";
import { workHref } from "@/lib/work-url";
import {
  earlierVersions,
  isLongNote,
  versionAnchor,
  versionDate,
  versionSummary,
  versionTitle,
} from "@/lib/work-versions";
import { VersionChanges } from "./VersionChanges";

export type HistoryOwner = {
  workName: string;
  canManage: boolean;
  isOwner: boolean;
  draftedChangesVersion: number;
};

/** Shows a version's notes, changes, downloads and owner controls. */
export function VersionDetail({
  workId,
  typeName,
  version,
  versions,
  current,
  download,
  onChanged,
  owner,
}: {
  workId: string;
  typeName: string;
  version: RecordedVersion;
  versions: RecordedVersion[];
  current: boolean;
  download: ReactNode;
  onChanged: () => void;
  owner: HistoryOwner;
}) {
  const earlier = earlierVersions(versions, version);
  const [baseline, setBaseline] = useState(earlier[0]?.number ?? 0);
  const against = earlier.find((one) => one.number === baseline);
  const withdrawn = Boolean(
    version.withdrawnAt || version.withdrawalExplanation,
  );

  return (
    <article aria-labelledby={versionAnchor(version)} className="min-w-0">
      <header className="flex flex-wrap items-start justify-between gap-x-8 gap-y-4">
        <div className="min-w-0">
          <h3
            className="flex flex-wrap items-baseline gap-x-3 font-display text-title font-medium tracking-[-0.02em] text-ink"
            id={versionAnchor(version)}
          >
            {versionTitle(version)}
            {current ? (
              <span className="text-meta font-medium text-accent">
                Published
              </span>
            ) : null}
            {withdrawn ? (
              <span className="text-meta font-medium text-stop">Withdrawn</span>
            ) : null}
          </h3>
          <p className="mt-1.5 flex flex-wrap items-baseline gap-x-3 gap-y-1 text-meta text-mute">
            <time dateTime={version.recordedAt}>{versionDate(version)}</time>
            {version.versionLabel ? (
              <span>{versionTag(version.versionLabel)}</span>
            ) : null}
            {version.notesEditedAt ? <span>Notes edited</span> : null}
          </p>
        </div>
        {download ? <div className="shrink-0">{download}</div> : null}
      </header>

      {withdrawn && !owner.isOwner ? (
        <div className="mt-6 max-w-[60ch]">
          <p className="font-prose text-prose text-mute">
            {version.withdrawalExplanation}
          </p>
          <p className="mt-3 text-meta text-mute">
            Its content, comparisons and downloads are unavailable.
          </p>
        </div>
      ) : (
        <>
          <p className="mt-6 max-w-[60ch] font-prose text-lede text-ink">
            {versionSummary(version, typeName)}
          </p>
          {version.notes ? <Note notes={version.notes} /> : null}

          <section
            aria-labelledby={`${versionAnchor(version)}-changes`}
            className="mt-8 border-t border-rule pt-6"
          >
            <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-3">
              <h4
                className="font-display text-section font-medium text-ink"
                id={`${versionAnchor(version)}-changes`}
              >
                What changed
              </h4>
              {against ? (
                <Baseline
                  chosen={baseline}
                  earlier={earlier}
                  onChoose={setBaseline}
                  version={version}
                />
              ) : null}
            </div>
            {against ? (
              <VersionChanges
                workId={workId}
                from={baseline}
                typeName={typeName}
                to={version.number}
              />
            ) : (
              <p className="mt-3 max-w-[60ch] text-meta text-mute">
                This is the first recorded version, so there is no earlier
                version to compare it with.
              </p>
            )}
          </section>
        </>
      )}

      {owner.canManage ? (
        <VersionManagement
          workId={workId}
          current={current}
          onChanged={onChanged}
          owner={owner}
          version={version}
        />
      ) : null}
    </article>
  );
}

function VersionManagement({
  workId,
  current,
  onChanged,
  owner,
  version,
}: {
  workId: string;
  current: boolean;
  onChanged: () => void;
  owner: HistoryOwner;
  version: RecordedVersion;
}) {
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
    <div className="mt-8 max-w-[38rem] border-rule border-t pt-2">
      <div className="flex min-h-control flex-wrap items-center gap-x-1">
        {!current ? (
          <Button
            aria-expanded={mode === "restore"}
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
          aria-expanded={mode === "correct"}
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
            aria-expanded={mode === "withdraw"}
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
            Your drafted changes become this version, images and layout
            included. Private prompts and allowed apps stay as they are.
          </p>
          <ActionRow
            busy={busy}
            confirm="Replace drafted changes"
            onCancel={() => setMode("")}
            onConfirm={() =>
              runMutation(
                () =>
                  restoreWorkVersion(
                    {
                      workId,
                      version: owner.draftedChangesVersion,
                    } satisfies Candidate,
                    workId,
                    version.number,
                  ),
                () => window.location.assign(workHref(workId, owner.workName)),
              )
            }
          />
        </div>
      ) : null}

      {mode === "correct" ? (
        <div className="grid gap-4 pt-4 pb-1">
          <Field label="Summary">
            <Input
              maxLength={200}
              onChange={(event) => setSummary(event.target.value)}
              value={summary}
            />
          </Field>
          <Field label="Notes">
            <Textarea
              maxLength={4000}
              onChange={(event) => setNotes(event.target.value)}
              value={notes}
            />
          </Field>
          <ActionRow
            busy={busy}
            confirm="Save notes"
            onCancel={() => setMode("")}
            onConfirm={() =>
              runMutation(
                () =>
                  correctWorkVersionNotes(workId, version.number, {
                    summary,
                    notes,
                  }),
                () => {
                  setMode("");
                  onChanged();
                },
              )
            }
          />
        </div>
      ) : null}

      {mode === "withdraw" ? (
        <div className="grid gap-4 pt-4 pb-1">
          <p className="text-meta text-mute">
            Readers see the version number, the date and this reason. Nobody can
            read or download it.
          </p>
          <Field label="Reason readers see">
            <Textarea
              maxLength={1000}
              onChange={(event) => setExplanation(event.target.value)}
              value={explanation}
            />
          </Field>
          <ActionRow
            busy={busy}
            confirm="Withdraw version"
            onCancel={() => setMode("")}
            onConfirm={() =>
              runMutation(
                () => withdrawWorkVersion(workId, version.number, explanation),
                () => {
                  setMode("");
                  onChanged();
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
    <p className="flex flex-wrap items-center gap-2 text-meta text-mute">
      <label htmlFor={field}>Since</label>
      <Select
        className="h-control-compact w-auto text-meta"
        id={field}
        onValueChange={(number) => onChoose(Number(number))}
        options={earlier.map((one) => ({
          value: String(one.number),
          label: `${versionTitle(one)} · ${versionDate(one)}`,
        }))}
        value={String(chosen)}
      />
    </p>
  );
}

function Note({ notes }: { notes: string }) {
  const foldable = isLongNote(notes);
  const [shown, setShown] = useState(!foldable);
  const id = useId();

  return (
    <Collapsible
      className="mt-4 max-w-[60ch]"
      onOpenChange={setShown}
      open={shown}
    >
      <p
        className={cn(
          "font-prose text-prose whitespace-pre-wrap text-mute",
          shown ? null : "line-clamp-4",
        )}
        id={id}
      >
        {notes}
      </p>
      {foldable ? (
        <CollapsibleTrigger asChild>
          <Button
            aria-controls={id}
            className="group/notes mt-1 -ml-2 px-2"
            size="compact"
            variant="ghost"
          >
            <ChevronRight
              aria-hidden="true"
              className="transition-transform duration-80 group-data-[state=open]/notes:rotate-90 motion-reduce:transition-none"
            />
            {shown ? "Show less" : "Read full notes"}
          </Button>
        </CollapsibleTrigger>
      ) : null}
    </Collapsible>
  );
}
