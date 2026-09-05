"use client";

import { CalendarClock, CircleAlert, Loader } from "lucide-react";
import { useState } from "react";
import { FormDialog } from "@/components/console/FormDialog";
import {
  cancelPostSchedule,
  readPostRevisions,
  replacePostSchedule,
} from "@/lib/api/posts";
import type { Post, PostRevision } from "@/lib/api/query";
import { readableMoment } from "@/lib/dates";
import { revisionWords } from "@/lib/post-history";
import { type LocalParts, localParts, toInstant } from "@/lib/schedule-time";
import styles from "./ScheduleBand.module.css";
import { ScheduleFields } from "./ScheduleFields";

export function ScheduleBand({
  post,
  onChanged,
  onFailure,
}: {
  post: Post;
  onChanged: (post: Post) => void;
  onFailure: (message: string) => void;
}) {
  const schedule = post.schedule;
  const [kept, setKept] = useState<PostRevision[] | null>(null);
  const [replacing, setReplacing] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [busy, setBusy] = useState(false);
  const [chosen, setChosen] = useState("");
  const [when, setWhen] = useState<LocalParts>({ date: "", time: "" });

  if (!schedule || schedule.state === "cancelled") return null;
  if (schedule.state === "published") return null;

  const running = schedule.state === "publishing";
  const stopped = schedule.state === "stopped";

  async function open() {
    if (!schedule) return;
    setChosen(schedule.revisionId);
    setWhen(localParts(schedule.at));
    setReplacing(true);
    if (kept) return;
    const answer = await readPostRevisions(post.id);
    if (answer.value) setKept(answer.value.revisions);
  }

  async function replace() {
    const at = toInstant(when.date, when.time);
    if (!at || !chosen) return;
    setBusy(true);
    const answer = await replacePostSchedule(post.id, chosen, at);
    setBusy(false);
    setReplacing(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onChanged(answer.value);
  }

  async function stop() {
    setBusy(true);
    const answer = await cancelPostSchedule(post.id);
    setBusy(false);
    setCancelling(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onChanged(answer.value);
  }

  return (
    <section
      aria-labelledby="going-live"
      className={styles.band}
      data-state={schedule.state}
    >
      <p className={styles.mark}>
        {stopped ? (
          <CircleAlert size={15} strokeWidth={1.9} aria-hidden="true" />
        ) : running ? (
          <Loader size={15} strokeWidth={1.9} aria-hidden="true" />
        ) : (
          <CalendarClock size={15} strokeWidth={1.9} aria-hidden="true" />
        )}
        <span id="going-live">{heading(schedule.state)}</span>
      </p>

      <p className={styles.said}>
        {stopped ? (
          schedule.stoppedBecause
        ) : running ? (
          <>Edition {schedule.revisionNumber} is going live now.</>
        ) : (
          <>
            Edition {schedule.revisionNumber} goes live{" "}
            <time dateTime={schedule.at}>{readableMoment(schedule.at)}</time>.
          </>
        )}
      </p>

      <p className={styles.meanwhile}>
        {stopped
          ? "Nothing was published. Schedule it again when you are ready."
          : post.status === "published"
            ? "Readers keep the edition on the blog now until then."
            : "Readers cannot see this post until then."}
      </p>

      {stopped || running ? null : (
        <div className={styles.actions}>
          <button
            className={styles.act}
            onClick={() => void open()}
            type="button"
          >
            Replace edition
          </button>
          <button
            className={styles.stop}
            onClick={() => setCancelling(true)}
            type="button"
          >
            Cancel
          </button>
        </div>
      )}

      <FormDialog
        busy={busy}
        commit="Replace"
        hint="The edition you leave stays in the history exactly as it is."
        onClose={() => setReplacing(false)}
        onCommit={() => void replace()}
        open={replacing}
        ready={chosen !== "" && toInstant(when.date, when.time) !== ""}
        title="What goes live instead?"
      >
        <fieldset className={styles.editions}>
          <legend>Edition</legend>
          {(kept ?? []).map((one) => (
            <label className={styles.edition} key={one.id}>
              <input
                checked={chosen === one.id}
                name="scheduled-edition"
                onChange={() => setChosen(one.id)}
                type="radio"
                value={one.id}
              />
              <span className={styles.number}>{one.number}</span>
              <span className={styles.editionName}>
                {revisionWords(one.capturedFor)}
                <span>{readableMoment(one.capturedAt)}</span>
              </span>
              {standingOf(one.id, schedule.revisionId, one.public) ? (
                <span className={styles.standing}>
                  {standingOf(one.id, schedule.revisionId, one.public)}
                </span>
              ) : null}
            </label>
          ))}
        </fieldset>
        <ScheduleFields id="replace-schedule" onChange={setWhen} parts={when} />
      </FormDialog>

      <FormDialog
        busy={busy}
        commit="Cancel the schedule"
        hint="The edition stays in the history and you can schedule it again."
        onClose={() => setCancelling(false)}
        onCommit={() => void stop()}
        open={cancelling}
        title="Stop this from going live?"
      >
        <p className={styles.confirm}>
          {post.title}
          <span>
            {post.status === "published"
              ? "Readers keep the edition on the blog now."
              : "The post stays a private draft."}
          </span>
        </p>
      </FormDialog>
    </section>
  );
}

// standingOf marks the edition readers have and the one already waiting.
function standingOf(
  id: string,
  scheduledId: string,
  onTheBlog: boolean,
): string {
  if (id === scheduledId) return "Waiting now";
  return onTheBlog ? "On the blog now" : "";
}

function heading(state: string): string {
  if (state === "stopped") return "Illarin stopped this schedule";
  if (state === "publishing") return "Going live";
  return "Waiting to go live";
}
