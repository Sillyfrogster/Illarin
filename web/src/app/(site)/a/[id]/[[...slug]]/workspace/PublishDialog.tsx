"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { ChangeList } from "@/components/changes/ChangeList";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { readDiscordChannel } from "@/lib/api/integrations";
import {
  compareDraftedChanges,
  fetchWaitingReplacement,
  fetchWork,
  publishWork,
  publishWorkVersion,
  type ReadinessItem,
  type VersionChangeGroup,
} from "@/lib/api/query";
import {
  DRAFTED_CHANGES_SAVED,
  useDraftedChanges,
} from "@/lib/drafted-changes";
import type { ReadinessTarget } from "@/lib/readiness";
import { Field, Note, Switch, TextAreaField, TextField } from "./fields";
import { ReadinessList } from "./ReadinessList";
import { useWorkspace } from "./state";

export function PublishDialog({
  typeName,
  onGo,
  unlisted,
  readiness,
}: {
  typeName: string;
  onGo: (target: ReadinessTarget) => void;
  unlisted: boolean;
  readiness: ReadinessItem[];
}) {
  const workspace = useWorkspace();
  const liveCandidate = useDraftedChanges();
  const [candidate] = useState(() => ({ version: liveCandidate.version }));
  const router = useRouter();
  const [groups, setGroups] = useState<VersionChangeGroup[] | null>(null);
  const [waiting, setWaiting] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [summary, setSummary] = useState("");
  const [notes, setNotes] = useState("");
  const [label, setLabel] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [stale, setStale] = useState(false);
  const [missing, setMissing] = useState<ReadinessItem[]>(
    readiness.filter((item) => !item.met),
  );
  const [notify, setNotify] = useState(true);
  const [discord, setDiscord] = useState(true);
  const [hasChannel, setHasChannel] = useState(false);

  useEffect(() => {
    if (workspace.isDraft || unlisted) return;
    const controller = new AbortController();
    void readDiscordChannel("account", controller.signal).then((answer) => {
      if (!controller.signal.aborted) {
        setHasChannel(Boolean(answer.value?.connected));
      }
    });
    return () => controller.abort();
  }, [workspace.isDraft, unlisted]);

  useEffect(() => {
    let active = true;
    void Promise.all([
      workspace.isDraft
        ? Promise.resolve(null)
        : compareDraftedChanges(candidate, workspace.workId),
      workspace.isDraft
        ? Promise.resolve(null)
        : fetchWaitingReplacement(workspace.workId),
      fetchWork(workspace.workId, undefined, true),
    ])
      .then(([changes, upload, work]) => {
        if (!active) return;
        if (!work)
          throw new Error(
            "Could not load your changes. Close this dialog and try again.",
          );
        setMissing(work.readiness?.filter((item) => !item.met) ?? []);
        setGroups(changes);
        setWaiting(Boolean(upload));
        setLoaded(true);
      })
      .catch((error) => {
        if (active)
          setMessage(
            error instanceof Error
              ? error.message
              : "Could not load your changes. Close this dialog and try again.",
          );
      });
    return () => {
      active = false;
    };
  }, [candidate, workspace.workId, workspace.isDraft]);

  async function publish() {
    if (busy) return;
    setBusy(true);
    setMessage("");
    try {
      const answer = workspace.isDraft
        ? await publishWork(candidate, workspace.workId)
        : await publishWorkVersion(candidate, workspace.workId, {
            discord: hasChannel && discord,
            notify,
            notes: notes.trim(),
            summary: summary.trim(),
            versionLabel: label.trim(),
          });
      if (answer.published) {
        liveCandidate.version = candidate.version;
        window.dispatchEvent(new Event(DRAFTED_CHANGES_SAVED));
        workspace.closePane();
        router.refresh();
        return;
      }
      setMessage(answer.error);
      setStale("code" in answer && answer.code === "drafted_changes_conflict");
      setMissing(answer.readiness?.filter((item) => !item.met) ?? []);
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "Could not publish. Try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !busy) workspace.closePane();
      }}
    >
      <DialogContent className="max-w-3xl">
        <div className="shrink-0 px-6 pt-6 pr-16 sm:px-8 sm:pt-8">
          <DialogTitle className="font-display text-section font-medium">
            Publish {workspace.isDraft ? `this ${typeName}` : "changes"}?
          </DialogTitle>
          <DialogDescription className="mt-2 text-ui text-mute">
            {workspace.isDraft
              ? "Anyone with the link can open your published page. Publishing cannot be undone."
              : "These changes become public. Earlier versions stay in history."}
          </DialogDescription>
        </div>
        <div className="min-h-0 overflow-y-auto px-6 py-6 sm:px-8">
          <div className="flex min-w-0 flex-col gap-6">
            {!loaded && !message ? (
              <output className="text-ui text-mute">
                Loading your changes…
              </output>
            ) : null}
            {groups ? (
              groups.length ? (
                <ChangeList groups={groups} />
              ) : (
                <Note>Nothing has changed since the last version.</Note>
              )
            ) : null}
            {waiting ? (
              <div className="flex flex-col gap-2">
                <Note>
                  Accept or discard your uploaded file before publishing.
                </Note>
                <Button
                  onClick={() => workspace.openPane({ kind: "replacement" })}
                >
                  Open uploaded file
                </Button>
              </div>
            ) : null}
            {missing.length > 0 ? (
              <ReadinessList items={missing} onGo={onGo} />
            ) : null}
            {!workspace.isDraft ? (
              <>
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

                <div className="flex flex-col gap-2">
                  <Switch
                    checked={notify}
                    hint="Anyone following it, or with it installed in a linked app, gets a notification."
                    label="Tell people following this"
                    onChange={setNotify}
                    pending={busy}
                  />
                  {hasChannel ? (
                    <Switch
                      checked={discord}
                      hint="The summary and a link go to the Discord channel in your settings."
                      label="Post to Discord"
                      onChange={setDiscord}
                      pending={busy}
                    />
                  ) : null}
                </div>
              </>
            ) : null}
            {message ? (
              <div
                className="flex flex-col gap-4 rounded-plate bg-stop-wash p-4"
                role="alert"
              >
                <p className="text-ui text-ink">{message}</p>
                {stale ? (
                  <>
                    <p className="text-meta text-mute">
                      What you wrote here is still here. Copy anything you want
                      to keep before reloading the page.
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
          </div>
        </div>
        <div className="flex shrink-0 justify-end gap-2 bg-deep px-6 py-4 sm:px-8">
          <Button disabled={busy} onClick={workspace.closePane} variant="ghost">
            Keep editing
          </Button>
          <Button
            disabled={
              !loaded ||
              waiting ||
              stale ||
              workspace.dirty ||
              workspace.busy ||
              (!workspace.isDraft && (!groups?.length || summary.trim() === ""))
            }
            loading={busy}
            onClick={publish}
            variant="primary"
          >
            {busy ? "Publishing…" : "Publish"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
