"use client";

import { Upload } from "lucide-react";
import { type ChangeEvent, useEffect, useRef, useState } from "react";
import { ChangeList } from "@/components/changes/ChangeList";
import { Button } from "@/components/ui/button";
import { RailBack } from "@/components/workspace/WorkspaceRail";
import {
  acceptWorkReplacement,
  cancelWorkReplacement,
  type IngestOperation,
  PromptsMadePublicError,
  type ReplacementDecision,
  readIngestOperation,
  uploadWorkReplacement,
  type VersionChangeGroup,
} from "@/lib/api/query";
import { useDraftedChanges } from "@/lib/drafted-changes";
import { replacementSubjectLabel } from "@/lib/replacement-subject";
import {
  replacementAction,
  replacementReady,
  unsettledReplacement,
} from "@/lib/work-publish";
import { MakePublicConfirmation } from "../MakePublicConfirmation";
import { Note } from "./fields";
import { ReplacementWarnings } from "./ReplacementWarnings";
import { useWorkspace } from "./state";

const POLL_MS = 600;

export function ReplacementStep({
  onApplied,
  onBack,
  onDiscarded,
  waiting,
  onWaiting,
}: {
  onApplied: (groups: VersionChangeGroup[]) => void;
  onBack: () => void;
  onDiscarded: () => void;
  waiting: IngestOperation | null;
  onWaiting: (operation: IngestOperation | null) => void;
}) {
  const workspace = useWorkspace();
  const candidate = useDraftedChanges();
  const fileInput = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [operation, setOperation] = useState<IngestOperation | null>(waiting);
  const [decisions, setDecisions] = useState<ReplacementDecision>({});
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [exposure, setExposure] = useState<string[] | null>(null);

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
    let polling = true;
    async function poll() {
      while (polling) {
        await new Promise((resolve) => setTimeout(resolve, POLL_MS));
        if (!polling) return;
        try {
          const next = await readIngestOperation(current.url);
          if (!polling) return;
          setOperation(next);
          onWaiting(unsettledReplacement(next));
          if (next.status !== "pending" && next.status !== "processing") return;
        } catch {
          setMessage(
            "Import status is unavailable. Reopen this panel to check again. Your published work has not changed.",
          );
          return;
        }
      }
    }
    void poll();
    return () => {
      polling = false;
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
    setExposure(null);
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
      if (error instanceof PromptsMadePublicError) {
        setExposure(error.prompts);
        return;
      }
      setExposure(null);
      setMessage(
        error instanceof Error
          ? error.message
          : "Illarin could not be reached. Check your connection and try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  function act(makePromptsPublic = false) {
    if (operation?.status === "failed") {
      beginAgain();
      return;
    }
    if (staged) {
      const groups = staged.preview.groups;
      void run(async () => {
        await acceptWorkReplacement(
          candidate,
          workspace.workId,
          staged.id,
          decisions,
          makePromptsPublic,
        );
        setExposure(null);
        onApplied(groups);
      });
      return;
    }
    if (!file) return;
    void run(async () => {
      setOperation(
        await uploadWorkReplacement(candidate, workspace.workId, file),
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
        Import a replacement file into your drafted changes. Review its changes
        before publishing a version.
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
            Read as {staged.preview.format}. New content stays private until you
            publish. Making private prompts public needs a separate confirmation
            because it can affect text already published.
          </p>
          <ReplacementWarnings preview={staged.preview} />
          {staged.preview.groups.length === 0 ? (
            <Note>This file matches your current content.</Note>
          ) : (
            <ChangeList groups={staged.preview.groups} />
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
                : "It must be the same type as this one"}
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
          onClick={() => act()}
          variant="primary"
        >
          {replacementAction(operation, busy)}
        </Button>
        {staged ? (
          <Button
            disabled={busy}
            onClick={() =>
              void run(async () => {
                await cancelWorkReplacement(workspace.workId, staged.id);
                onDiscarded();
              })
            }
            variant="ghost"
          >
            Discard this file
          </Button>
        ) : null}
      </div>
      {exposure ? (
        <MakePublicConfirmation
          prompts={exposure}
          keepsAPrivatePrompt={
            staged !== null && staged.preview.privatePrompts > 0
          }
          pending={busy}
          replacement
          onKeepPrivate={() => setExposure(null)}
          onMakePublic={() => act(true)}
        />
      ) : null}
    </div>
  );
}
