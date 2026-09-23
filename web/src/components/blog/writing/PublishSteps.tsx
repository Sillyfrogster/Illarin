"use client";

import { useCallback, useEffect, useState } from "react";
import { CheckRow } from "@/components/ui/check-row";
import { Field, TextArea } from "@/components/ui/field";
import { readWorkspace } from "@/lib/api/blog";
import {
  cancelPostSchedule,
  publishPost,
  readPostRevisions,
  recoverPost,
  replacePostSchedule,
  republishPost,
  schedulePost,
  unpublishPost,
} from "@/lib/api/posts";
import type { Post, PostRevision } from "@/lib/api/query";
import { readableMoment } from "@/lib/dates";
import { useBlogAddress } from "@/lib/origins";
import {
  atLeastAnHourAhead,
  type LocalParts,
  localParts,
  toInstant,
} from "@/lib/schedule-time";
import { ScheduleFields } from "./ScheduleFields";
import { Commit, Editions, Heading } from "./StepParts";

const SAID_LIMIT = 500;

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
  const blogAddress = useBlogAddress();
  const [hasChannel, setHasChannel] = useState(false);
  const [busy, setBusy] = useState(false);
  const [when, setWhen] = useState<LocalParts>(() => atLeastAnHourAhead());
  const [discord, setDiscord] = useState(true);
  const live = post.status === "published";
  const at = toInstant(when.date, when.time);
  const ready = door === "now" || at !== "";

  useEffect(() => {
    let active = true;
    void readWorkspace().then((answer) => {
      if (active) setHasChannel(Boolean(answer.value?.discord));
    });
    return () => {
      active = false;
    };
  }, []);

  async function commit() {
    setBusy(true);
    const version = await onSaveFirst();
    const announcement = { discord: hasChannel && discord };
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
        line={publishHint(door, post, `${blogAddress}/${post.slug}`)}
        title={
          door === "later"
            ? "Schedule this post"
            : live
              ? "Publish the changes"
              : "Publish this post"
        }
      />
      {door === "later" ? (
        <ScheduleFields id="publish-schedule" onChange={setWhen} parts={when} />
      ) : null}
      {hasChannel && !post.publishedAt ? (
        <CheckRow checked={discord} onChange={setDiscord}>
          Post to the blog's Discord
        </CheckRow>
      ) : null}
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

export function UnpublishStep({ onFailure, onSettled, post }: StepProps) {
  const [busy, setBusy] = useState(false);
  const [reason, setReason] = useState("");
  const [explanation, setExplanation] = useState("");

  async function commit() {
    setBusy(true);
    const answer = await unpublishPost(
      post.id,
      post.version,
      reason.trim(),
      explanation.trim(),
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
        line="Readers lose access. The post and its history stay, so you can republish it later."
        title="Unpublish this post?"
      />
      <Field
        hint="Illarin keeps this with the post. Readers never see it."
        htmlFor="unpublishing-reason"
        label="Private unpublishing reason"
      >
        <TextArea
          id="unpublishing-reason"
          maxLength={SAID_LIMIT}
          onChange={(event) => setReason(event.target.value)}
          rows={3}
          value={reason}
        />
      </Field>
      <Field
        hint="Shown on the post's address. Leave it empty and readers get the general message alone."
        htmlFor="unpublishing-explanation"
        label="Public explanation"
      >
        <TextArea
          id="unpublishing-explanation"
          maxLength={SAID_LIMIT}
          onChange={(event) => setExplanation(event.target.value)}
          rows={3}
          value={explanation}
        />
      </Field>
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready={reason.trim() !== ""}
        tone="stop"
        word="Unpublish post"
      />
    </>
  );
}

export function RepublishStep({ onFailure, onSettled, post }: StepProps) {
  const kept = useRevisions(post.id, onFailure);
  const [busy, setBusy] = useState(false);
  const [chosen, setChosen] = useState(post.publicRevisionId ?? "");

  async function commit() {
    if (!chosen) return;
    setBusy(true);
    const answer = await republishPost(post.id, post.version, chosen);
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
        line="It returns at the same address with its original publication date."
        title="Choose a revision to republish"
      />
      <Editions
        chosen={chosen}
        name="returning-edition"
        onChoose={setChosen}
        revisions={kept}
        standingOf={(one) => (one.public ? "Previously published" : "")}
      />
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready={chosen !== ""}
        word="Republish post"
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
      onFailure(answer.error ?? "The post could not be restored. Try again.");
      return;
    }
    onSettled(answer.value);
  }

  return (
    <>
      <Heading
        line="It comes back as it was before you deleted it."
        title="Restore this post?"
      />
      {deletion ? (
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 font-prose text-meta">
          <dt className="text-mute">Deleted</dt>
          <dd className="text-ink">
            {readableMoment(deletion.at)} by @{deletion.by}
          </dd>
          <dt className="text-mute">Permanently deleted</dt>
          <dd className="text-ink">{readableMoment(deletion.until)}</dd>
        </dl>
      ) : null}
      <Commit
        busy={busy}
        onCommit={() => void commit()}
        ready
        word="Restore post"
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
        line="The previously scheduled revision remains in history."
        title="Choose a scheduled revision"
      />
      <Editions
        chosen={chosen}
        name="scheduled-edition"
        onChoose={setChosen}
        revisions={kept}
        standingOf={(one) =>
          one.id === schedule?.revisionId
            ? "Currently scheduled"
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
      onFailure(answer.error ?? "The post is still scheduled.");
      return;
    }
    onSettled(answer.value);
  }

  return (
    <>
      <Heading
        line="The revision stays in history and can be scheduled again."
        title="Cancel scheduled publication?"
      />
      {schedule ? (
        <p className="font-prose text-meta text-ink">
          Revision {schedule.revisionNumber}, due {readableMoment(schedule.at)}.{" "}
          <span className="text-mute">
            {post.status === "published"
              ? "Readers keep the currently published revision."
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

function useRevisions(
  postId: string,
  onFailure: (message: string) => void,
): PostRevision[] | null {
  const [kept, setKept] = useState<PostRevision[] | null>(null);

  const load = useCallback(async () => {
    const answer = await readPostRevisions(postId);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "Revisions could not be loaded. Try again.");
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

function publishHint(
  door: "now" | "later",
  post: Post,
  address: string,
): string {
  const schedule = post.schedule;
  const waiting =
    schedule?.state === "pending" || schedule?.state === "publishing";
  if (door === "now" && waiting) {
    return "Publishing now cancels the scheduled publication.";
  }
  if (post.status === "published") {
    return "Publishing or scheduling saves a revision of your current writing. Later edits do not change that revision.";
  }
  return `It goes live at ${address}. After that, only an admin can change the address or the name on it.`;
}
