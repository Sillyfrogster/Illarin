"use client";

import { Upload } from "lucide-react";
import { type ChangeEvent, useEffect, useRef, useState } from "react";
import { FormDialog } from "@/components/console/FormDialog";
import {
  acceptAssetReplacement,
  cancelAssetReplacement,
  type IngestOperation,
  type ReplacementDecision,
  readIngestOperation,
  uploadAssetReplacement,
} from "@/lib/api/query";
import {
  replacementSubjectLabel,
  summariseReplacement,
} from "@/lib/replacement-subject";
import { useWorkingCopy } from "@/lib/working-copy";
import styles from "./ReplaceFileDialog.module.css";

/** How often the dialog asks what Illarin has made of the file so far. */
const POLL_MS = 600;

/** Uploading a replacement, reading what it carries, and deciding whether to take it. */
export function ReplaceFileDialog({
  assetId,
  waiting,
  onClose,
  onWaiting,
  onAccepted,
  onDiscarded,
}: {
  assetId: string;
  waiting: IngestOperation | null;
  onClose: () => void;
  onWaiting: (operation: IngestOperation | null) => void;
  onAccepted: () => void;
  onDiscarded: () => void;
}) {
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
          onWaiting(stillWaiting(next));
          if (next.status !== "pending" && next.status !== "processing") return;
        } catch {
          setMessage(
            "Illarin lost sight of this file. Your published page is untouched; reopen this to check again.",
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

  const commit = commitLabel(operation, busy);

  function act() {
    if (operation?.status === "failed") {
      beginAgain();
      return;
    }
    if (staged) {
      void run(async () => {
        await acceptAssetReplacement(candidate, assetId, staged.id, decisions);
        onAccepted();
      });
      return;
    }
    if (!file) return;
    void run(async () => {
      setOperation(await uploadAssetReplacement(candidate, assetId, file));
    });
  }

  return (
    <FormDialog
      open
      title="Replace this file"
      hint="The file decides. Everything it carries replaces what is on this page, and what it does not carry stays as it is."
      commit={commit}
      busy={busy || reading}
      ready={commitReady(operation, file, decisions)}
      onClose={onClose}
      onCommit={act}
      destructive={
        staged ? (
          <button
            type="button"
            className={styles.discard}
            onClick={() =>
              void run(async () => {
                await cancelAssetReplacement(assetId, staged.id);
                onDiscarded();
              })
            }
          >
            Discard this file
          </button>
        ) : null
      }
    >
      {reading ? (
        <p className={styles.state} aria-live="polite">
          Illarin is reading your file. Readers keep the published version while
          it works, and you can leave this open.
        </p>
      ) : null}

      {operation?.status === "failed" && operation.failure ? (
        <p className={styles.failure} role="alert">
          {operation.failure.message}
        </p>
      ) : null}

      {staged ? (
        <div className={styles.preview}>
          <p className={styles.format}>
            Read as {staged.preview.format}. Nothing below reaches readers until
            you publish an update.
          </p>

          {staged.preview.changes.length === 0 ? (
            <p className={styles.state}>
              This file carries the same content your page already holds.
            </p>
          ) : (
            <ul className={styles.changes}>
              {summariseReplacement(staged.preview.changes).map((part) => (
                <li key={part.subject}>
                  <strong>{part.label}</strong>
                  <span>{part.detail}</span>
                  {part.replacesYourEdit ? (
                    <span data-conflict="true">Replaces an edit of yours</span>
                  ) : null}
                </li>
              ))}
            </ul>
          )}

          {unrepresentable.length > 0 ? (
            <fieldset className={styles.decisions}>
              <legend>Content this file cannot hold</legend>
              <p>
                This file has no place for these. Say whether to keep each one
                on the page or let the file take it away.
              </p>
              {unrepresentable.map((role) => (
                <div key={role} className={styles.decision}>
                  <span>{replacementSubjectLabel(role)}</span>
                  <div className={styles.choice}>
                    {(["keep", "remove"] as const).map((answer) => (
                      <label key={answer}>
                        <input
                          type="radio"
                          name={`decision-${role}`}
                          checked={decisions[role] === answer}
                          onChange={() =>
                            setDecisions((current) => ({
                              ...current,
                              [role]: answer,
                            }))
                          }
                        />
                        {answer === "keep" ? "Keep it" : "Remove it"}
                      </label>
                    ))}
                  </div>
                </div>
              ))}
            </fieldset>
          ) : null}
        </div>
      ) : null}

      {!operation ? (
        <div className={styles.fileField}>
          <input
            ref={fileInput}
            id="replacement-file"
            type="file"
            onChange={choose}
          />
          <label htmlFor="replacement-file">
            <Upload size={22} strokeWidth={1.35} aria-hidden="true" />
            <span>{file ? file.name : "Choose the replacement file"}</span>
            <small>
              {file
                ? "Choose a different file"
                : "It must be the same kind of asset as this one"}
            </small>
          </label>
        </div>
      ) : null}

      {message ? (
        <p className={styles.failure} role="alert">
          {message}
        </p>
      ) : null}
    </FormDialog>
  );
}

/** The operation the page still has to account for, and nothing once it is settled. */
function stillWaiting(operation: IngestOperation): IngestOperation | null {
  const waiting = ["pending", "processing", "preview"];
  return waiting.includes(operation.status) ? operation : null;
}

/** What the dialog's one action does in the state it is in. */
function commitLabel(operation: IngestOperation | null, busy: boolean): string {
  if (operation?.status === "preview") {
    return busy ? "Applying…" : "Apply this file";
  }
  if (operation?.status === "failed") return "Choose another file";
  if (operation) return "Reading…";
  return busy ? "Uploading…" : "Upload this file";
}

/** Whether that action has everything it needs. */
function commitReady(
  operation: IngestOperation | null,
  file: File | null,
  decisions: ReplacementDecision,
): boolean {
  if (operation?.status === "preview") {
    return (operation.preview?.unrepresentable ?? []).every(
      (role) => decisions?.[role] !== undefined,
    );
  }
  if (operation?.status === "failed") return true;
  return !operation && file !== null;
}
