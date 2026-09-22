"use client";

import { SiDiscord } from "@icons-pack/react-simple-icons";
import { Bell } from "lucide-react";
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
import { LineLink } from "@/components/ui/line-link";
import { readDiscordChannel } from "@/lib/api/integrations";
import {
  compareDraftedChanges,
  fetchWaitingReplacement,
  fetchWork,
  publishWork,
  publishWorkVersion,
  type ReadinessItem,
  type VersionChangeGroup,
  type WorkDetail,
} from "@/lib/api/query";
import {
  DRAFTED_CHANGES_SAVED,
  useDraftedChanges,
} from "@/lib/drafted-changes";
import type { ReadinessTarget } from "@/lib/readiness";
import { Field, Note, TextAreaField, TextField } from "./fields";
import { Hearer, HearerCheck, PublishSubject } from "./PublishParts";
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
  const [work, setWork] = useState<WorkDetail | null>(null);

  useEffect(() => {
    if (unlisted) return;
    const controller = new AbortController();
    void readDiscordChannel("account", controller.signal).then((answer) => {
      if (!controller.signal.aborted) {
        setHasChannel(Boolean(answer.value?.connected));
      }
    });
    return () => controller.abort();
  }, [unlisted]);

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
        setWork(work);
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
        ? await publishWork(candidate, workspace.workId, hasChannel && discord)
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

  const next = (work?.latestVersion?.number ?? 0) + 1;
  const hearers = !workspace.isDraft || !unlisted;

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !busy) workspace.closePane();
      }}
    >
      <DialogContent className="max-w-4xl">
        <div className="grid min-h-0 flex-1 md:grid-cols-[16rem_minmax(0,1fr)]">
          <aside className="shrink-0 bg-inset px-6 py-5 pr-16 md:p-7">
            <PublishSubject
              from={workspace.isDraft ? "Draft" : `Version ${next - 1}`}
              to={
                workspace.isDraft
                  ? unlisted
                    ? "Unlisted"
                    : "Public"
                  : `Version ${next}`
              }
              work={work}
            />
          </aside>

          <div className="flex min-h-0 min-w-0 flex-col">
            <div className="shrink-0 px-6 pt-6 sm:px-8 sm:pt-8 md:pr-16">
              <DialogTitle className="font-display text-section font-medium">
                {workspace.isDraft
                  ? `Publish this ${typeName}?`
                  : "Publish your changes?"}
              </DialogTitle>
              <DialogDescription className="mt-2 max-w-[52ch] text-ui text-mute">
                {workspace.isDraft
                  ? unlisted
                    ? "Anyone with the link can open it. Publishing can't be undone."
                    : "It shows in Browse and on your profile. Publishing can't be undone."
                  : "Readers get the new version. Earlier versions stay in history."}
              </DialogDescription>
            </div>

            <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6 sm:px-8">
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
                      onClick={() =>
                        workspace.openPane({ kind: "replacement" })
                      }
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

                    <div className="grid gap-6 sm:grid-cols-[minmax(0,1fr)_10rem]">
                      <Field hint="optional" label="Notes">
                        <TextAreaField
                          maxLength={4000}
                          onChange={(event) => setNotes(event.target.value)}
                          rows={3}
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
                    </div>
                  </>
                ) : null}
                {hearers ? (
                  <section
                    aria-labelledby="publish-hearers"
                    className="flex flex-col gap-1"
                  >
                    <h3
                      className="font-ui text-meta font-medium text-mute"
                      id="publish-hearers"
                    >
                      Who hears about it
                    </h3>
                    {!workspace.isDraft ? (
                      <Hearer
                        control="publish-notify"
                        icon={<Bell aria-hidden="true" />}
                        line="Followers and linked apps with it installed get a notification."
                        title="Notify followers"
                      >
                        <HearerCheck
                          checked={notify}
                          id="publish-notify"
                          disabled={busy}
                          onChange={setNotify}
                        />
                      </Hearer>
                    ) : null}
                    {unlisted ? null : (
                      <Hearer
                        control={hasChannel ? "publish-discord" : undefined}
                        icon={<SiDiscord aria-hidden="true" />}
                        line={
                          hasChannel
                            ? workspace.isDraft
                              ? "Your channel gets its name, blurb and a link."
                              : "Your channel gets the summary and a link."
                            : "No channel connected yet."
                        }
                        title="Post to Discord"
                      >
                        {hasChannel ? (
                          <HearerCheck
                            checked={discord}
                            id="publish-discord"
                            disabled={busy}
                            onChange={setDiscord}
                          />
                        ) : (
                          <LineLink
                            className="text-accent hover:text-accent"
                            href="/settings#discord-channel"
                          >
                            Connect
                          </LineLink>
                        )}
                      </Hearer>
                    )}
                  </section>
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
                          What you wrote here is still here. Copy anything you
                          want to keep before reloading the page.
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

            <div className="flex shrink-0 flex-wrap justify-end gap-2 px-6 pt-2 pb-6 sm:px-8 sm:pb-8">
              <Button
                disabled={busy}
                onClick={workspace.closePane}
                variant="ghost"
              >
                Keep editing
              </Button>
              <Button
                disabled={
                  !loaded ||
                  waiting ||
                  stale ||
                  workspace.dirty ||
                  workspace.busy ||
                  (!workspace.isDraft &&
                    (!groups?.length || summary.trim() === ""))
                }
                loading={busy}
                onClick={publish}
                variant="primary"
              >
                {busy
                  ? "Publishing…"
                  : workspace.isDraft
                    ? `Publish ${typeName}`
                    : `Publish version ${next}`}
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
