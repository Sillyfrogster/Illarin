"use client";

import { useCallback, useEffect, useState } from "react";
import { Field, TextArea } from "@/components/ui/field";
import {
  cancelPostSchedule,
  deletePost,
  publishPost,
  readPostRevisions,
  recoverPost,
  replacePostSchedule,
  republishPost,
  schedulePost,
  withdrawPost,
} from "@/lib/api/posts";
import type { Post, PostRevision } from "@/lib/api/query";
import { readableMoment } from "@/lib/dates";
import {
  atLeastAnHourAhead,
  type LocalParts,
  localParts,
  toInstant,
} from "@/lib/schedule-time";
import { AnnouncementChoice } from "./AnnouncementChoice";
import { ScheduleFields } from "./ScheduleFields";
import { Commit, Editions, Heading, Subject } from "./StepParts";

/** How long a takedown reason or a reader's explanation may run. */
const SAID_LIMIT = 500;

/** What every step of the rail is handed: the post, and the ways back out of it. */
export type StepProps = {
  onFailure: (message: string) => void;
  onSaveFirst: () => Promise<number>;
  onSettled: (post: Post) => void;
  post: Post;
};

export function PublishStep({
  door,
  onFailure,
  onSaveFirst,
  onSettled,
  post,
}: StepProps & { door: "now" | "later" }) {
  const [busy, setBusy] = useState(false);
  const [when, setWhen] = useState<LocalParts>(() => atLeastAnHourAhead());
  const [sending, setSending] = useState<string[] | null>(null);
  const [pinging, setPinging] = useState<string[]>([]);
  const [note, setNote] = useState("");
  const live = post.status === "published";
  const at = toInstant(when.date, when.time);
  const ready = door === "now" || at !== "";

  async function commit() {
    setBusy(true);
    const version = await onSaveFirst();
    const announcement = {
      destinationIds: sending ?? undefined,
      roleDestinationIds: pinging.length > 0 ? pinging : undefined,
      note: note.trim() || undefined,
    };
    const answer =
      door === "later"
        ? await schedulePost(post.id, version, at, announcement)
        : await publishPost(post.id, version, announcement);
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "The post was not published.");
      return;
    }
    onSettled(answer.value);
  }

  return (
    <>
      <Heading
        line={publishHint(door, post)}
        title={
          door === "later"
            ? "Schedule this post"
            : live
              ? "Publish the changes"
              : "Publish this post"
        }
      />
      <Subject post={post} />
      {door === "later" ? (
        <ScheduleFields id="publish-schedule" onChange={setWhen} parts={when} />
      ) : null}
      <AnnouncementChoice
        announced={Boolean(post.publishedAt)}
        chosen={sending}
        note={note}
        onChosen={setSending}
        onNote={setNote}
        onPinging={setPinging}
        pinging={pinging}
        postId={post.id}
        transition={live ? "changes" : "publish"}
      />
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready={ready}
        word={
          door === "later" ? "Schedule" : live ? "Publish changes" : "Publish"
        }
      />
    </>
  );
}

export function WithdrawStep({ onFailure, onSettled, post }: StepProps) {
  const [busy, setBusy] = useState(false);
  const [reason, setReason] = useState("");
  const [explanation, setExplanation] = useState("");
  const [sending, setSending] = useState<string[] | null>(null);
  const [note, setNote] = useState("");

  async function commit() {
    setBusy(true);
    const answer = await withdrawPost(
      post.id,
      post.version,
      reason.trim(),
      explanation.trim(),
      { destinationIds: sending, note },
    );
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "The post is still on the blog.");
      return;
    }
    onSettled(answer.value);
  }

  return (
    <>
      <Heading
        line="Its editions, pictures and dates stay. You can put it back at the same address."
        title="Take this post out of public view"
      />
      <Subject post={post} />
      <Field
        hint="Illarin keeps this with the post. Readers never see it."
        htmlFor="withdrawal-reason"
        label="Why it is coming down"
      >
        <TextArea
          id="withdrawal-reason"
          maxLength={SAID_LIMIT}
          onChange={(event) => setReason(event.target.value)}
          rows={3}
          value={reason}
        />
      </Field>
      <Field
        hint="Shown on the post's address. Leave it empty and readers get the general message alone."
        htmlFor="withdrawal-explanation"
        label="What readers see"
      >
        <TextArea
          id="withdrawal-explanation"
          maxLength={SAID_LIMIT}
          onChange={(event) => setExplanation(event.target.value)}
          rows={3}
          value={explanation}
        />
      </Field>
      <AnnouncementChoice
        announced
        chosen={sending}
        note={note}
        onChosen={setSending}
        onNote={setNote}
        onPinging={() => undefined}
        pinging={[]}
        postId={post.id}
        transition="withdraw"
      />
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready={reason.trim() !== ""}
        tone="stop"
        word="Take it down"
      />
    </>
  );
}

export function RepublishStep({ onFailure, onSettled, post }: StepProps) {
  const kept = useRevisions(post.id, onFailure);
  const [busy, setBusy] = useState(false);
  const [chosen, setChosen] = useState(post.publicRevisionId ?? "");
  const [sending, setSending] = useState<string[] | null>(null);
  const [note, setNote] = useState("");

  async function commit() {
    if (!chosen) return;
    setBusy(true);
    const answer = await republishPost(post.id, post.version, chosen, {
      destinationIds: sending,
      note,
    });
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "The post is still out of public view.");
      return;
    }
    onSettled(answer.value);
  }

  return (
    <>
      <Heading
        line="It returns to the same address under the date it first published."
        title="Which edition do readers get?"
      />
      <Editions
        chosen={chosen}
        name="returning-edition"
        onChoose={setChosen}
        revisions={kept}
        standingOf={(one) => (one.public ? "Was on the blog" : "")}
      />
      <AnnouncementChoice
        announced
        chosen={sending}
        note={note}
        onChosen={setSending}
        onNote={setNote}
        onPinging={() => undefined}
        pinging={[]}
        postId={post.id}
        transition="republish"
      />
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready={chosen !== ""}
        word="Put it back"
      />
    </>
  );
}

export function DeleteStep({ onFailure, onSettled, post }: StepProps) {
  const [busy, setBusy] = useState(false);

  async function commit() {
    setBusy(true);
    const answer = await deletePost(post.id, post.version);
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "The post is still here.");
      return;
    }
    onSettled(answer.value);
  }

  return (
    <>
      <Heading
        line="Illarin holds it for thirty days. After that the writing, the editions and the pictures are gone."
        title="Delete this post?"
      />
      <Subject post={post} />
      <p className="font-prose text-meta text-mute">
        {post.publishedAt ? "Published, then taken down." : "Never published."}
      </p>
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready
        tone="stop"
        word="Delete it"
      />
    </>
  );
}

export function RecoverStep({ onFailure, onSettled, post }: StepProps) {
  const [busy, setBusy] = useState(false);
  const deletion = post.deletion;

  async function commit() {
    setBusy(true);
    const answer = await recoverPost(post.id, post.version);
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "The post could not be brought back.");
      return;
    }
    onSettled(answer.value);
  }

  return (
    <>
      <Heading
        line="It returns exactly as it was, in the standing it was deleted from."
        title="Bring this post back"
      />
      {deletion ? (
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 font-prose text-meta">
          <dt className="text-mute">Deleted</dt>
          <dd className="text-ink">
            {readableMoment(deletion.at)} by @{deletion.by}
          </dd>
          <dt className="text-mute">Gone for good</dt>
          <dd className="text-ink">{readableMoment(deletion.until)}</dd>
        </dl>
      ) : null}
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready
        word="Bring it back"
      />
    </>
  );
}

export function RescheduleStep({ onFailure, onSettled, post }: StepProps) {
  const kept = useRevisions(post.id, onFailure);
  const schedule = post.schedule;
  const [busy, setBusy] = useState(false);
  const [chosen, setChosen] = useState(schedule?.revisionId ?? "");
  const [when, setWhen] = useState<LocalParts>(() =>
    schedule ? localParts(schedule.at) : atLeastAnHourAhead(),
  );
  const at = toInstant(when.date, when.time);

  async function commit() {
    if (!at || !chosen) return;
    setBusy(true);
    const answer = await replacePostSchedule(post.id, chosen, at);
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "The schedule did not change.");
      return;
    }
    onSettled(answer.value);
  }

  return (
    <>
      <Heading
        line="The edition you leave stays in the history exactly as it is."
        title="What goes live instead?"
      />
      <Editions
        chosen={chosen}
        name="scheduled-edition"
        onChoose={setChosen}
        revisions={kept}
        standingOf={(one) =>
          one.id === schedule?.revisionId
            ? "Waiting now"
            : one.public
              ? "On the blog now"
              : ""
        }
      />
      <ScheduleFields id="replace-schedule" onChange={setWhen} parts={when} />
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready={chosen !== "" && at !== ""}
        word="Replace"
      />
    </>
  );
}

export function UnscheduleStep({ onFailure, onSettled, post }: StepProps) {
  const [busy, setBusy] = useState(false);
  const schedule = post.schedule;

  async function commit() {
    setBusy(true);
    const answer = await cancelPostSchedule(post.id);
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "The schedule is still standing.");
      return;
    }
    onSettled(answer.value);
  }

  return (
    <>
      <Heading
        line="The edition stays in the history and you can schedule it again."
        title="Stop this from going live?"
      />
      {schedule ? (
        <p className="font-prose text-meta text-ink">
          Edition {schedule.revisionNumber}, due {readableMoment(schedule.at)}.{" "}
          <span className="text-mute">
            {post.status === "published"
              ? "Readers keep the edition on the blog now."
              : "The post stays a private draft."}
          </span>
        </p>
      ) : null}
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready
        tone="stop"
        word="Cancel the schedule"
      />
    </>
  );
}

// useRevisions reads the kept editions once, for the steps that choose between them.
function useRevisions(
  postId: string,
  onFailure: (message: string) => void,
): PostRevision[] | null {
  const [kept, setKept] = useState<PostRevision[] | null>(null);

  const load = useCallback(async () => {
    const answer = await readPostRevisions(postId);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "The editions could not be read.");
      setKept([]);
      return;
    }
    setKept(answer.value.revisions);
  }, [postId, onFailure]);

  useEffect(() => {
    void load();
  }, [load]);

  return kept;
}

// publishHint says what this step will do, including to a schedule already waiting.
function publishHint(door: "now" | "later", post: Post): string {
  const schedule = post.schedule;
  const waiting =
    schedule?.state === "pending" || schedule?.state === "publishing";
  if (door === "now" && waiting) {
    return "Publishing now stops the edition waiting to go live.";
  }
  if (post.status === "published") {
    return "The version in front of you is captured either way, and later edits do not change it.";
  }
  return "This fixes the address and puts your name on the post. Only an admin can change either afterwards.";
}
