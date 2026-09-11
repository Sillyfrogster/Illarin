"use client";

import { Upload } from "lucide-react";
import { type ChangeEvent, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { RailBack } from "@/components/workspace/WorkspaceRail";
import {
  acceptAssetReplacement,
  cancelAssetReplacement,
  type IngestOperation,
  type ReplacementDecision,
  readIngestOperation,
  uploadAssetReplacement,
} from "@/lib/api/query";
import {
  replacementAction,
  replacementReady,
  unsettledReplacement,
} from "@/lib/asset-publication";
import {
  type ReplacementSummary,
  replacementSubjectLabel,
  summariseReplacement,
} from "@/lib/replacement-subject";
import { useWorkingCopy } from "@/lib/working-copy";
import { Note } from "./fields";
import { useWorkspace } from "./state";

const POLL_MS = 600;

export function ReplacementStep({
  onApplied,
  onBack,
  onDiscarded,
  waiting,
  onWaiting,
}: {
  onApplied: (changes: ReplacementSummary[]) => void;
  onBack: () => void;
  onDiscarded: () => void;
  waiting: IngestOperation | null;
  onWaiting: (operation: IngestOperation | null) => void;
}) {
  const workspace = useWorkspace();
  const candidate = useWorkingCopy();
  const fileInput = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [operation, setOperation] = useState<IngestOperation | null>(waiting);
  const [decisions, setDecisions] = useState<ReplacementDecision>({});
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  const reading =
    operation?.status === "pending" || operation?.status === "processing";
  const staged =
    operation?.status === "preview" && operation.preview
      ? { id: operation.id, preview: operation.preview }
      : null;
  const unrepresentable = staged?.preview.unrepresentable ?? [];

  useEffect(() => {
    if (!operation || !reading) return;
    const current = operation;
    let watching = true;
    async function poll() {
      while (watching) {
        await new Promise((resolve) => setTimeout(resolve, POLL_MS));
        if (!watching) return;
        try {
          const next = await readIngestOperation(current.url);
          if (!watching) return;
          setOperation(next);
          onWaiting(unsettledReplacement(next));
          if (next.status !== "pending" && next.status !== "processing") return;
        } catch {
          setMessage(
            "Import status is unavailable. Reopen this panel to check again. Your published asset has not changed.",
          );
          return;
        }
      }
    }
    void poll();
    return () => {
      watching = false;
    };
  }, [operation, reading, onWaiting]);

  function choose(event: ChangeEvent<HTMLInputElement>) {
    setFile(event.target.files?.[0] ?? null);
    setMessage("");
  }

  function beginAgain() {
    setOperation(null);
    setFile(null);
    setMessage("");
    onWaiting(null);
    if (fileInput.current) fileInput.current.value = "";
  }

  async function run(work: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setMessage("");
    try {
      await work();
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "Illarin could not be reached. Check your connection and try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  function act() {
    if (operation?.status === "failed") {
      beginAgain();
      return;
    }
    if (staged) {
      const changes = summariseReplacement(staged.preview.changes);
      void run(async () => {
        await acceptAssetReplacement(
          candidate,
          workspace.assetId,
          staged.id,
          decisions,
        );
        onApplied(changes);
      });
      return;
    }
    if (!file) return;
    void run(async () => {
      setOperation(
        await uploadAssetReplacement(candidate, workspace.assetId, file),
      );
    });
  }

  return (
    <div className="flex flex-col gap-5">
      <RailBack onClick={onBack}>Publication</RailBack>
      <h3 className="font-display text-section font-medium text-ink">
        Replace the file
      </h3>
      <Note>
        Import a replacement file into your working copy. Review its changes
        before publishing an update.
      </Note>

      {reading ? (
        <p aria-live="polite" className="text-ui text-ink">
          Illarin is reading your file. Readers keep the published version while
          it works, and you can leave this open.
        </p>
      ) : null}

      {operation?.status === "failed" && operation.failure ? (
        <p
          className="rounded-control bg-stop-wash p-3 text-meta text-ink"
          role="alert"
        >
          {operation.failure.message}
        </p>
      ) : null}

      {staged ? (
        <div className="flex flex-col gap-5">
          <p className="text-ui text-ink">
            Read as {staged.preview.format}. Nothing here reaches readers until
            you publish an update.
          </p>
          {staged.preview.changes.length === 0 ? (
            <Note>This file matches your current content.</Note>
          ) : (
            <ReplacementChanges
              changes={summariseReplacement(staged.preview.changes)}
            />
          )}
          {unrepresentable.length > 0 ? (
            <fieldset className="min-w-0 border-0 p-0">
              <legend className="mb-2 font-display text-ui font-medium text-ink">
                Content this file cannot hold
              </legend>
              <Note>
                Choose whether to keep or remove existing content that this
                format cannot store.
              </Note>
              <div className="mt-4 flex flex-col gap-4">
                {unrepresentable.map((role) => (
                  <div className="flex flex-col gap-2" key={role}>
                    <p className="text-ui text-ink" id={`subject-${role}`}>
                      {replacementSubjectLabel(role)}
                    </p>
                    <div
                      aria-labelledby={`subject-${role}`}
                      className="flex flex-wrap gap-2"
                      role="radiogroup"
                    >
                      {(["keep", "remove"] as const).map((answer) => (
                        <label
                          className="inline-flex min-h-11 cursor-pointer items-center gap-2 rounded-control bg-deep px-4 text-meta text-ink has-checked:bg-accent-wash"
                          key={answer}
                        >
                          <input
                            checked={decisions[role] === answer}
                            className="size-4 accent-[var(--v-action)]"
                            name={`decision-${role}`}
                            onChange={() =>
                              setDecisions((current) => ({
                                ...current,
                                [role]: answer,
                              }))
                            }
                            type="radio"
                          />
                          {answer === "keep" ? "Keep it" : "Remove it"}
                        </label>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </fieldset>
          ) : null}
        </div>
      ) : null}

      {operation ? null : (
        <div>
          <input
            accept="*"
            className="sr-only peer"
            id="replacement-file"
            onChange={choose}
            ref={fileInput}
            type="file"
          />
          <label
            className="flex cursor-pointer flex-col items-center gap-2 rounded-plate bg-deep px-5 py-8 text-center hover:bg-rule/45 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-accent peer-focus-visible:outline-offset-3"
            htmlFor="replacement-file"
          >
            <Upload aria-hidden="true" size={22} strokeWidth={1.35} />
            <span className="text-ui font-medium text-ink wrap-anywhere">
              {file ? file.name : "Choose the replacement file"}
            </span>
            <span className="text-meta text-mute">
              {file
                ? "Choose a different file"
                : "It must be the same kind of asset as this one"}
            </span>
          </label>
        </div>
      )}

      {message ? (
        <p
          className="rounded-control bg-stop-wash p-3 text-meta text-ink"
          role="alert"
        >
          {message}
        </p>
      ) : null}

      <div className="flex flex-wrap items-center gap-2">
        <Button
          disabled={!replacementReady(operation, file, decisions) || reading}
          loading={busy}
          onClick={act}
          variant="primary"
        >
          {replacementAction(operation, busy)}
        </Button>
        {staged ? (
          <Button
            disabled={busy}
            onClick={() =>
              void run(async () => {
                await cancelAssetReplacement(workspace.assetId, staged.id);
                onDiscarded();
              })
            }
            variant="ghost"
          >
            Discard this file
          </Button>
        ) : null}
      </div>
    </div>
  );
}

export function ReplacementChanges({
  changes,
}: {
  changes: ReplacementSummary[];
}) {
  return (
    <ul className="flex list-none flex-col gap-3">
      {changes.map((part) => (
        <li key={part.subject}>
          <span className="text-ui font-medium text-ink">{part.label}</span>
          <span className="mt-0.5 block text-meta text-mute">
            {part.detail}
          </span>
          {part.replacesYourEdit ? (
            <span className="mt-1 inline-block rounded-control bg-stop-wash px-2 py-0.5 text-label font-medium text-ink">
              Overwrites an existing edit
            </span>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
