"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { RailBack } from "@/components/workspace/WorkspaceRail";
import { publishAssetUpdate, type ReadinessItem } from "@/lib/api/query";
import type { ReadinessTarget } from "@/lib/readiness";
import type { ReplacementSummary } from "@/lib/replacement-subject";
import { useWorkingCopy } from "@/lib/working-copy";
import { Field, Note, TextAreaField, TextField } from "./fields";
import { ReadinessList } from "./ReadinessList";
import { ReplacementChanges } from "./ReplacementStep";
import { useWorkspace } from "./state";

/** Where the reviewed working copy becomes the next public version readers get. */
export function ReviewStep({
  applied,
  kind,
  onBack,
  onGo,
  onPublished,
}: {
  applied: ReplacementSummary[] | null;
  kind: string;
  onBack: () => void;
  onGo: (target: ReadinessTarget) => void;
  onPublished: () => void;
}) {
  const workspace = useWorkspace();
  const candidate = useWorkingCopy();
  const [summary, setSummary] = useState("");
  const [notes, setNotes] = useState("");
  const [label, setLabel] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [stale, setStale] = useState(false);
  const [missing, setMissing] = useState<ReadinessItem[]>([]);

  async function publish() {
    setBusy(true);
    setMessage("");
    setStale(false);
    setMissing([]);
    const answer = await publishAssetUpdate(candidate, workspace.assetId, {
      notes: notes.trim(),
      summary: summary.trim(),
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
    <div className="flex flex-col gap-5">
      <RailBack onClick={onBack}>Publication</RailBack>
      <h3 className="font-display text-section font-medium text-ink">
        Review this update
      </h3>
      <Note>
        Readers get the page beside this rail exactly as it stands, working copy{" "}
        {candidate.version}. The version they have now keeps its place in the
        history.
      </Note>

      {applied && applied.length > 0 ? (
        <section className="flex flex-col gap-3">
          <h4 className="font-display text-ui font-medium text-ink">
            What the file you applied changed
          </h4>
          <ReplacementChanges changes={applied} />
        </section>
      ) : null}

      <Field
        hint="readers see this in the history"
        label="Summary of what changed"
      >
        <TextField
          autoComplete="off"
          maxLength={200}
          onChange={(event) => setSummary(event.target.value)}
          placeholder="Rewrote her opening and added two greetings"
          value={summary}
        />
      </Field>

      <Field hint="optional" label="Notes">
        <TextAreaField
          maxLength={4000}
          onChange={(event) => setNotes(event.target.value)}
          rows={4}
          value={notes}
        />
      </Field>

      <Field hint="optional" label="Version">
        <TextField
          autoComplete="off"
          maxLength={60}
          onChange={(event) => setLabel(event.target.value)}
          placeholder="v2.1"
          value={label}
        />
      </Field>

      <Note>
        Illarin sends this update nowhere. Choosing who hears about an update
        comes later, and nothing here announces anything.
      </Note>

      {message ? (
        <div
          className="flex flex-col gap-4 rounded-plate bg-stop-wash p-4"
          role="alert"
        >
          <p className="text-ui text-ink">{message}</p>
          {missing.length > 0 ? (
            <ReadinessList items={missing} onGo={onGo} />
          ) : null}
          {stale ? (
            <>
              <p className="text-meta text-mute">
                What you wrote here is still here. Copy anything you want to
                keep before reloading the page.
              </p>
              <Button
                className="self-start"
                onClick={() => window.location.reload()}
              >
                Reload the page
              </Button>
            </>
          ) : null}
        </div>
      ) : null}

      <Button
        className="self-start"
        disabled={summary.trim() === ""}
        loading={busy}
        onClick={publish}
        variant="primary"
      >
        {busy ? "Publishing…" : `Publish this ${kind} update`}
      </Button>
    </div>
  );
}
