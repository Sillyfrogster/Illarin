"use client";

import { useCallback, useState } from "react";
import { ChangeList } from "@/components/changes/ChangeList";
import { Button } from "@/components/ui/button";
import {
  type AnnouncementChoice,
  NO_CHOICE,
  WorkAnnouncementChoice,
} from "@/components/updates/WorkAnnouncementChoice";
import { RailBack } from "@/components/workspace/WorkspaceRail";
import {
  publishWorkVersion,
  type ReadinessItem,
  type VersionChangeGroup,
} from "@/lib/api/query";
import { useDraftedChanges } from "@/lib/drafted-changes";
import type { ReadinessTarget } from "@/lib/readiness";
import { Field, Note, TextAreaField, TextField } from "./fields";
import { ReadinessList } from "./ReadinessList";
import { useWorkspace } from "./state";

export function ReviewStep({
  applied,
  typeName,
  onBack,
  onGo,
  onPublished,
  unlisted,
}: {
  applied: VersionChangeGroup[] | null;
  typeName: string;
  onBack: () => void;
  onGo: (target: ReadinessTarget) => void;
  onPublished: () => void;
  unlisted: boolean;
}) {
  const workspace = useWorkspace();
  const candidate = useDraftedChanges();
  const [summary, setSummary] = useState("");
  const [notes, setNotes] = useState("");
  const [label, setLabel] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [stale, setStale] = useState(false);
  const [missing, setMissing] = useState<ReadinessItem[]>([]);
  const [announcement, setAnnouncement] =
    useState<AnnouncementChoice>(NO_CHOICE);
  const [needsConsent, setNeedsConsent] = useState(false);
  const chooseAnnouncement = useCallback((choice: AnnouncementChoice) => {
    setAnnouncement(choice);
    setNeedsConsent(false);
  }, []);

  async function publish() {
    setBusy(true);
    setMessage("");
    setStale(false);
    setMissing([]);
    const answer = await publishWorkVersion(candidate, workspace.workId, {
      announceUnlisted: announcement.announceUnlisted,
      integrationIds: announcement.integrationIds ?? undefined,
      notify: announcement.notify,
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
    setStale(answer.code === "drafted_changes_conflict");
    setNeedsConsent(answer.field === "announceUnlisted");
    setMissing(answer.readiness?.filter((item) => !item.met) ?? []);
  }

  return (
    <div className="flex flex-col gap-5">
      <RailBack onClick={onBack}>Publication</RailBack>
      <h3 className="font-display text-section font-medium text-ink">
        Review this version
      </h3>
      <Note>
        Publishing makes these changes public. Drafted changes{" "}
        {candidate.version}. The current published version remains in version
        history.
      </Note>

      {applied && applied.length > 0 ? (
        <section className="flex flex-col gap-3">
          <Note>These changes came from the file you applied.</Note>
          <ChangeList groups={applied} />
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

      <WorkAnnouncementChoice
        workId={workspace.workId}
        choice={announcement}
        disabled={busy}
        needsConsent={needsConsent}
        onChange={chooseAnnouncement}
        unlisted={unlisted}
      />

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
        {busy ? "Publishing…" : `Publish this ${typeName} version`}
      </Button>
    </div>
  );
}
