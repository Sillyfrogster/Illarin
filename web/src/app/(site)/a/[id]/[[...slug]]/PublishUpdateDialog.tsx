"use client";

import { useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { publishAssetUpdate, type ReadinessItem } from "@/lib/api/query";
import { readinessHref } from "@/lib/readiness";
import { useWorkingCopy } from "@/lib/working-copy";
import styles from "./PublishUpdateDialog.module.css";

/** The form that hands the reviewed working copy to readers as the next public version. */
export function PublishUpdateDialog({
  assetId,
  kind,
  changed,
  replacementWaiting,
  onClose,
  onPublished,
}: {
  assetId: string;
  kind: string;
  changed: boolean;
  replacementWaiting: boolean;
  onClose: () => void;
  onPublished: () => void;
}) {
  const candidate = useWorkingCopy();
  const [summary, setSummary] = useState("");
  const [notes, setNotes] = useState("");
  const [label, setLabel] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [stale, setStale] = useState(false);
  const [missing, setMissing] = useState<ReadinessItem[]>([]);

  const blocked = blockedReason(changed, replacementWaiting);

  async function publish() {
    setBusy(true);
    setMessage("");
    setStale(false);
    setMissing([]);
    const answer = await publishAssetUpdate(candidate, assetId, {
      summary: summary.trim(),
      notes: notes.trim(),
      versionLabel: label.trim(),
    });
    setBusy(false);
    if (answer.published) {
      onPublished();
      return;
    }
    setMessage(answer.error);
    setStale(answer.code === "working_copy_conflict");
    setMissing(answer.readiness?.filter((item) => !item.met) ?? []);
  }

  return (
    <FormDialog
      open
      title={`Publish this ${kind} update`}
      hint="Readers get the page exactly as you see it now. The version they have keeps its place in the history."
      commit="Publish update"
      busy={busy}
      ready={!blocked && summary.trim() !== ""}
      onClose={onClose}
      onCommit={publish}
    >
      <p className={styles.candidate}>
        Publishing the page as you see it now, working copy {candidate.version}.
      </p>

      {blocked ? <p className={styles.blocked}>{blocked}</p> : null}

      <Field
        label="Summary"
        htmlFor="update-summary"
        hint="One line saying what changed. Every update needs one, and readers see it in the history."
      >
        <input
          id="update-summary"
          value={summary}
          maxLength={200}
          placeholder="Rewrote her opening and added two greetings"
          autoComplete="off"
          onChange={(event) => setSummary(event.target.value)}
        />
      </Field>

      <Field
        label="Notes"
        htmlFor="update-notes"
        hint="The longer explanation, where one is worth writing. Optional."
      >
        <textarea
          id="update-notes"
          value={notes}
          maxLength={4000}
          rows={4}
          onChange={(event) => setNotes(event.target.value)}
        />
      </Field>

      <Field
        label="Version"
        htmlFor="update-label"
        hint="Your own name for this version. Illarin never reads it, and repeating one is allowed. Optional."
      >
        <input
          id="update-label"
          value={label}
          maxLength={60}
          placeholder="v2.1"
          autoComplete="off"
          onChange={(event) => setLabel(event.target.value)}
        />
      </Field>

      <p className={styles.announcements}>
        Illarin sends this update nowhere. Choosing who hears about an update
        comes later, and nothing here announces anything.
      </p>

      {message ? (
        <div className={styles.refusal} role="alert">
          <p>{message}</p>
          {missing.length > 0 ? (
            <ul>
              {missing.map((item) => (
                <li key={item.id}>
                  <strong>{item.label}</strong> {item.detail}{" "}
                  <a href={readinessHref(item)} onClick={onClose}>
                    take me there
                  </a>
                </li>
              ))}
            </ul>
          ) : null}
          {stale ? (
            <p className={styles.kept}>
              What you wrote here is still here. Copy anything you want to keep
              before reloading the page.
            </p>
          ) : null}
        </div>
      ) : null}
    </FormDialog>
  );
}

/** Why publishing cannot start yet, and nothing where it can. */
function blockedReason(changed: boolean, replacementWaiting: boolean): string {
  if (replacementWaiting) {
    return "An uploaded file is waiting for your review. Accept or discard it before publishing.";
  }
  if (!changed) {
    return "Nothing has changed since the last update. Edit the page or replace the file first.";
  }
  return "";
}
